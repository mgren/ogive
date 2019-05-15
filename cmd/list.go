package cmd

import (
	"fmt"
	"github.com/InVisionApp/tabular"
	"github.com/awnumar/memguard"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/mgren/ogive/crypt"
	"github.com/mgren/ogive/object"
	"github.com/mgren/ogive/profile"
	"github.com/mgren/ogive/util"
	"github.com/spf13/cobra"
	"os"
)

func init() {
	rootCmd.AddCommand(listCmd)

	tab = tabular.New()
	tab.Col("SIZE", "SIZE", 10)
	tab.Col("DATE", "DATE", 12)
	tab.Col("STAT", "STATUS", 6)
	tab.Col("ID", "STORAGE ID", 10)
	tab.Col("NAME", "FILENAME", 8)
}

var tab tabular.Table

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List archives.",
	Long:  "Lists all ogive archives in bucket. Lists entire bucket and HEADs each file.",
	Run: func(cmd *cobra.Command, args []string) {
		inner, err := profile.Open(profileFile)
		if err != nil {
			util.Fail(err, "Failed to open profile. Wrong password?")
		}

		gcm, err := crypt.GetGCM(inner.Key, 32)
		inner.Key.Destroy()
		if err != nil {
			util.Fail(err, "Failed to set up decryptors.")
		}

		svc := s3.New(util.GetSession(inner))
		format := tab.Print(tabular.All)

		err = svc.ListObjectsV2Pages(&s3.ListObjectsV2Input{
			Bucket: &inner.BucketName,
		}, func(page *s3.ListObjectsV2Output, lastPage bool) bool {
			for _, key := range page.Contents {
				res, err := svc.HeadObject(&s3.HeadObjectInput{
					Bucket: &inner.BucketName,
					Key:    key.Key,
				})
				if err != nil {
					fmt.Fprintln(os.Stderr, "Failed to head object", *key.Key, err)
					continue
				}

				// Just to make the list show less clutter in case the bucket is not ogive-exclusive
				if *res.ContentType != "application/x-ogive" {
					continue
				}

				obj, err := object.Parse(res, key.Key, gcm, nil)
				if err != nil {
					fmt.Fprintln(os.Stderr, "Invalid file metadata", *key.Key, err)
					continue
				}

				// https://golang.org/src/time/format.go
				fmt.Printf(format,
					util.SizeIEC(int64(obj.Size)),
					obj.LastModified.Format("2006-Jan-02"),
					obj.Restore, *key.Key, obj.Name)

			}
			return !lastPage
		})

		if err != nil {
			util.Fail(err, "Failed to list bucket.")
		}

		memguard.SafeExit(0)
	},
}
