package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version is the version of repository-mapper
//
// Don't forget to tag a new version: git tag -a 0.2.0 -m "xyz feature released in this tag"
// then: git push origin 0.2.0
const Version = "0.4.0"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Outputs the current version.",
	Long:  "Outputs the current version.",
	Run:   version,
}

func version(_ *cobra.Command, _ []string) {
	fmt.Printf("%s\n", Version)
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
