package cmd

import (
	"fmt"
	"github.com/awnumar/memguard"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/mgren/ogive/crypt"
	"github.com/mgren/ogive/object"
	"github.com/mgren/ogive/profile"
	"github.com/mgren/ogive/progress"
	"github.com/mgren/ogive/util"
	"github.com/spf13/cobra"
)

func init() {
	getCmd.Flags().StringVarP(&output, "output", "o", "", "Override destination filename.")
	rootCmd.AddCommand(getCmd)
}

var output string

var getCmd = &cobra.Command{
	Use:   "get <source_file> <destination_directory>",
	Short: "Download file.",
	Long:  "Download and decrypt file, saving it under the original filename.",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		inner, err := profile.Open(profileFile)
		if err != nil {
			util.Fail(err, "Failed to open profile. Wrong password?")
		}

		gcm, err := crypt.GetGCM(inner.Key, 32)
		if err != nil {
			util.Fail(err, "Failed to set up decryptors.")
		}

		sess := util.GetSession(inner)
		svc := s3.New(sess)

		res, err := svc.HeadObject(&s3.HeadObjectInput{
			Bucket: &inner.BucketName,
			Key:    &args[0],
		})
		if err != nil {
			util.Fail(err, "Failed to head object.")
		}

		obj, err := object.Parse(res, &args[0], gcm, inner.Key)
		inner.Key.Destroy()
		if err != nil {
			util.Fail(err, "Invalid file metadata")
		}
		if obj.Restore != "READY" {
			util.Fail(err, "File not restored, please run ogive restore first.")
		}

		if output == "" {
			output = obj.Name
		}

		fmt.Println("File will be saved as", output)

		writer, err := crypt.GetCryptWriter(obj.Key, args[1], output)
		if err != nil {
			util.Fail(err, "Failed to open file for writing.")
		}
		// This writer is initiated with 2 bytes already written.
		// Since sio supports two different ciphers, it lazily initiates only one of them when it knows which one
		// i.e. when the second byte is written to the writer.
		// The destruction must be delayed until that happens, otherwise the underlying AES asm code will run into a memory violation during key expansion.
		// Since there is no out-of-the-box way to notify this routine of when that happens, the writer is initialized via magic.
		// Both sio version and cipher are pinned for the sio.EncryptReader, so all uploaded files will always have the same header.
		// The first two bytes (0x20 0x00) are written manually using WriterAtFake which then omits first two bytes on the very first call.
		// As bad as it sounds, it relies on exported constants, it's just that they weren't supposed to be used this way.
		obj.Key.Destroy()
		defer writer.Close()

		proxyWriter := progress.NewWriter(writer)
		done := make(chan bool)
		go progress.TrackProgress(&proxyWriter, int(*res.ContentLength)-2, done)

		// WriterAtFake works, because s3manager.Downloader.Concurrency = 1 assures sequential write.
		// Using actual WriterAt with AES is technically possible but requires every At to be a multiple of block size.
		// This could also be implemented with an intermediate buffer of size s3manager.Download.Concurreny * s3manager.Download.PartSize,
		// but Download doesn't guarantee a write of size s3manager.Download.PartSize even for non-final parts, which makes it much more difficult to do.
		// Best way would be probably to request with s3.GetObjectInput.Range specified and implement concurrency locally.
		fake := util.NewWriterAtFake(&proxyWriter)

		_, err = s3manager.NewDownloader(sess, func(u *s3manager.Downloader) {
			u.PartSize = 50 << 20
			u.Concurrency = 1
		}).Download(&fake, &s3.GetObjectInput{
			Bucket: &inner.BucketName,
			Key:    &args[0],
		})
		if err != nil {
			util.Fail(err, "Failed to download file.")
		}

		<-done
		fmt.Printf("Successfully downloaded %s as %s. Exiting...\n", args[0], output)

		// This is needed because memguard.SafeExit relies on os.Exit, which doesn't honour defer stack.
		writer.Close()
		memguard.SafeExit(0)
	},
}
