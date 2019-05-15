package cmd

import (
	"fmt"
	"github.com/awnumar/memguard"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/mgren/ogive/crypt"
	"github.com/mgren/ogive/object"
	"github.com/mgren/ogive/profile"
	"github.com/mgren/ogive/progress"
	"github.com/mgren/ogive/util"
	"github.com/spf13/cobra"
	"path/filepath"
)

func init() {
	rootCmd.AddCommand(putCmd)
}

var putCmd = &cobra.Command{
	Use:   "put <source_file>",
	Short: "Upload file.",
	Long:  "Encrypt and upload file to S3 Glacier Deep Archive.",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		inner, err := profile.Open(profileFile)
		if err != nil {
			util.Fail(err, "Failed to open profile. Wrong password?")
		}

		base := filepath.Base(args[0])

		obj, err := object.Prepare(inner.Key, base)
		if err != nil {
			util.Fail(err, "Failed to prepare file for encryption.")
		}

		reader, size, err := crypt.GetCryptReader(obj.Key, args[0])
		if err != nil {
			util.Fail(err, "Failed to encrypt file.")
		}

		fmt.Printf("Uploading %s as %s\n", base, obj.Name)

		proxyReader := progress.NewReader(reader)
		done := make(chan bool)
		go progress.TrackProgress(&proxyReader, size, done)

		_, err = s3manager.NewUploader(util.GetSession(inner), func(u *s3manager.Uploader) {
			u.PartSize = util.GetPartSize(int64(size))
		}).Upload(&s3manager.UploadInput{
			Body:        &proxyReader,
			Bucket:      &inner.BucketName,
			Key:         &obj.Name,
			ContentType: aws.String("application/x-ogive"),
			StorageClass: aws.String("DEEP_ARCHIVE"),
			Metadata: map[string]*string{
				"Nonce": aws.String(fmt.Sprintf("%x", obj.Nonce)),
			},
		})

		if err != nil {
			util.Fail(err, "Failed to upload file.")
		}

		<-done
		fmt.Printf("Successfully uploaded %s as %s\n", base, obj.Name)
		memguard.SafeExit(0)
	},
}
