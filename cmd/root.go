package cmd

import (
	"github.com/mgren/ogive/util"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&profileFile, "profile", "p", util.GetDefaultProfileLoc(), "Location of ogive profile file.")
	cobra.MarkFlagFilename(rootCmd.PersistentFlags(), "profile")
}

var profileFile string

var rootCmd = &cobra.Command{
	Use:   "ogive",
	Short: "secure backups with AWS S3 Glacier Deep Archive",
	Long:  "ogive is a simple commandline tool for storing and retrieving cryptographically secure backups from AWS S3 Glacier Deep Archive.",
}

// Execute is the hook for main to start Cobra
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		util.Fail(err, "Critical error.")
	}
}
