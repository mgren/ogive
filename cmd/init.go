package cmd

import (
	"errors"
	"fmt"
	"github.com/awnumar/memguard"
	"github.com/mgren/ogive/input"
	"github.com/mgren/ogive/profile"
	"github.com/mgren/ogive/util"
	"github.com/spf13/cobra"
	"os"
)

func init() {
	initCmd.Flags().BoolVarP(&reinit, "reinit", "r", false, "Reinitialize an existing profile to change password and/or AWS keys. Old profile is stored as \"<name>.bak\".")
	rootCmd.AddCommand(initCmd)
}

var reinit bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Set up an ogive profile.",
	Long:  "Set up an ogive profile, including your cryptographic key and S3 bucket location.",
	Run: func(cmd *cobra.Command, args []string) {
		var profileInner *profile.InnerData
		var err error

		if reinit {
			profileInner, err = profile.Open(profileFile)
		} else {
			profileInner, err = profile.NewInner()
		}
		if err != nil {
			util.Fail(err, "Failed to generate profile.")
		}
		defer profileInner.Key.Destroy()

		pwd, err := input.GetMaskedInput("Enter password", "", "", 64, 8)
		if err != nil {
			util.Fail(err, "Failed to read password.")
		}
		defer pwd.Destroy()

		pwd2, err := input.GetMaskedInput("Confirm password", "", "", 64, 8)
		if err != nil {
			util.Fail(err, "Failed to read password.")
		}
		defer pwd2.Destroy()

		assertEqual(pwd, pwd2)

		id, secret := getMaskedInputs(reinit)

		if id != nil {
			profileInner.AWSKeyId = id
		}

		if secret != nil {
			profileInner.AWSSecret = secret
		}

		defer profileInner.AWSKeyId.Destroy()
		defer profileInner.AWSSecret.Destroy()

		if reinit {
			err = os.Rename(profileFile, profileFile+".bak")
			if err != nil {
				util.Fail(err, "Failed to back up profile.")
			}
		} else {
			// This remains unchanged on reinit
			profileInner.BucketName, profileInner.Region, profileInner.Endpoint = getInputs()
		}

		err = profile.Save(pwd, profileInner, profileFile)
		if err != nil {
			util.Fail(err, "Failed to generate profile.")
		}

		fmt.Println("Profile successfully created.")
		memguard.SafeExit(0)
	},
}

func assertEqual(a, b *memguard.LockedBuffer) {
	eq, err := memguard.Equal(a, b)
	if err != nil {
		util.Fail(err, "Error reading password.")
	}
	if !eq {
		util.Fail(errors.New("Passwords must match."), "Passwords must match.")
	}
	b.Destroy()
}

func getInputs() (bucket, region, endpoint string) {
	var err error
	var buf *memguard.LockedBuffer

	// https://docs.aws.amazon.com/AmazonS3/latest/dev/BucketRestrictions.html
	buf, err = input.GetInput("Enter AWS S3 bucket name", "", "", 63, 3)
	if err != nil {
		util.Fail(err, "Failed to generate profile.")
	}
	bucket = string(buf.Buffer())

	buf, err = input.GetInput("Enter AWS S3 Region", "eu-west-1", "", 64, 0)
	if err != nil {
		util.Fail(err, "Failed to generate profile.")
	}
	region = string(buf.Buffer())

	buf, err = input.GetInput("Enter AWS S3 endpoint", "https://s3."+region+".amazonaws.com", "", 64, 0)
	if err != nil {
		util.Fail(err, "Failed to generate profile.")
	}
	endpoint = string(buf.Buffer())

	return
}

func getMaskedInputs(allowBlank bool) (id, secret *memguard.LockedBuffer) {
	var err error
	min := 1

	if allowBlank {
		min = 0
	}

	id, err = input.GetMaskedInput("Enter AWS Key ID", "", "", 64, min)
	if err != nil {
		util.Fail(err, "Failed to generate profile.")
	}

	secret, err = input.GetMaskedInput("Enter AWS Key Secret", "", "", 64, min)
	if err != nil {
		util.Fail(err, "Failed to generate profile.")
	}

	return
}
