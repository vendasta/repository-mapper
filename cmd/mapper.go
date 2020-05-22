package cmd

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"github.com/libgit2/git2go"
	"github.com/spf13/cobra"
)

var (
	branchName  string
	org         string
	script      string
	makePr      bool
	title       string
	description string
	noFetch     bool
	workspace   string
)

func init() {
	usr, _ := user.Current()
	workspace = filepath.Join(usr.HomeDir, "repository-mapper")

	rootCmd.Flags().StringVarP(&branchName, "branch-name", "b", "", "The branch to create. Should be globally unique.")
	rootCmd.MarkFlagRequired("branch-name")

	rootCmd.Flags().StringVarP(&org, "org", "o", "vendasta", "The github organization the repos live in.")

	rootCmd.Flags().StringVarP(&script, "script", "s", "", "Path to the script to run in each repository")
	rootCmd.MarkFlagRequired("script")

	rootCmd.Flags().StringVarP(&title, "title", "t", "", "Title of the PR")
	rootCmd.Flags().StringVarP(&description, "description", "d", "", "Description of the PR")

}

var rootCmd = &cobra.Command{
	Use:   "repository-mapper",
	Short: "Run scripts on repositories across your org",
	Long:  `Run scripts and queries on repositories across your org`,
	Args:  cobra.MinimumNArgs(1),
	Run:   run,
}

func run(cmd *cobra.Command, args []string) {
	err := validateArgs()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	fmt.Println(script)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func validateArgs() error {
	_, err := os.Stat(script)
	if os.IsNotExist(err) {
		return fmt.Errorf("Could not find script: '%s'", script)
	}
	script, err = filepath.Abs(script)
	if err != nil {
		return err
	}

	if makePr {
		if title == "" {
			return fmt.Errorf("A PR title is required")
		}
		if description == "" {
			return fmt.Errorf("A PR description is required")
		}
	}

	return nil
}

func isDir(p string) bool {
	info, err := os.Stat(p)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func checkoutRepo(repo string) {
	repoPath := filepath.Join(workspace, repo)
	if !isDir(repoPath) {
		cloneRepo(repo, repoPath)
	}
}

func cloneRepo(repo string, dest string) {
	git2go.Clone()
}
