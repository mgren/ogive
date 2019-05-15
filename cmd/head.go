package cmd

import (
	"fmt"
	"github.com/awnumar/memguard"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/mgren/ogive/object"
	"github.com/mgren/ogive/profile"
	"github.com/mgren/ogive/util"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(headCmd)
}

var headCmd = &cobra.Command{
	Use:   "head <storage_id>",
	Short: "Head a specific file.",
	Long:  "Head a specific ogive file on S3 and retrieve its current archival status. Prints out file status and exits with code: 0 - file available for download, 1 - error occurred, 2 - file not available for download.",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		inner, err := profile.Open(profileFile)
		if err != nil {
			util.Fail(err, "Failed to open profile. Wrong password?")
		}
		inner.Key.Destroy() // Not needed here

		svc := s3.New(util.GetSession(inner))

		res, err := svc.HeadObject(&s3.HeadObjectInput{
			Bucket: &inner.BucketName,
			Key:    &args[0],
		})
		if err != nil {
			util.Fail(err, "Failed to head object.")
		}

		obj, err := object.Parse(res, nil, nil, nil)
		if err != nil {
			util.Fail(err, "Failed to parse response.")
		}

		fmt.Println(obj.Restore)

		if obj.Restore != "READY" {
			memguard.SafeExit(2)
		}

		memguard.SafeExit(0)
	},
}
