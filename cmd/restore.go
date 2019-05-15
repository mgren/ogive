package cmd

import (
	"fmt"
	"github.com/awnumar/memguard"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/mgren/ogive/profile"
	"github.com/mgren/ogive/util"
	"github.com/spf13/cobra"
)

func init() {
	restoreCmd.Flags().IntVarP(&lifetime, "lifetime", "t", 1, "Specifies the number of days to retain the restored object before returning it to Deep Archive.")
	rootCmd.AddCommand(restoreCmd)
}

var lifetime int

var restoreCmd = &cobra.Command{
	Use:   "restore <storage_id>",
	Short: "Restore a specific file.",
	Long:  "Initiate file recovery from Deep Archive. Bulk Restore is used. Use \"head\" command to verify when the file becomes ready for download.",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		inner, err := profile.Open(profileFile)
		if err != nil {
			util.Fail(err, "Failed to open profile. Wrong password? Exiting...")
			memguard.SafeExit(1)
		}
		inner.Key.Destroy() // Not needed here

		svc := s3.New(util.GetSession(inner))

		_, err = svc.RestoreObject(&s3.RestoreObjectInput{
			Bucket: &inner.BucketName,
			Key:    &args[0],
			RestoreRequest: &s3.RestoreRequest{
				Days: aws.Int64(int64(lifetime)),
				GlacierJobParameters: &s3.GlacierJobParameters{
					Tier: aws.String("Bulk"),
				},
			},
		})

		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				case s3.ErrCodeObjectAlreadyInActiveTierError:
					fmt.Println("Restoration already completed.")
				case "RestoreAlreadyInProgress": // Doesn't seem to be defined in current version of AWS SDK
					fmt.Println("Restoration already in progress.")
				default:
					util.Fail(err, "Failed to request restore.")
				}
			} else {
				util.Fail(err, "Failed to request restore.")
			}
		} else {
			fmt.Println("Restoration request sent.")
		}

		memguard.SafeExit(0)
	},
}
