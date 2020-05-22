package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/golang/glog"
	// git "github.com/libgit2/git2go/v30"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"

	"github.com/spf13/cobra"
	// "github.com/libgit2/git2go/v30"
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

	skipExitCode = 10
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

	glog.Infof("Using script: %s", script)

	allResults := map[string]*runResults{}

	for _, repoName := range args {
		results, err := runRepo(repoName)
		if err != nil {
			glog.Errorf("%s: %s", repoName, err.Error())
			continue
		}
		logResults(results)
		allResults[repoName] = results
	}
	summarizeResults(allResults)
	saveResults(allResults)
}

func summarizeResults(allResults map[string]*runResults) {
	var successes, skips, failures []*runResults
	for _, result := range allResults {
		switch result.ExitCode {
		case 0:
			successes = append(successes, result)
		case skipExitCode:
			skips = append(skips, result)
		default:
			failures = append(failures, result)
		}
	}

	glog.Info("===============")
	glog.Info("✅ SUCCEEDED ✅")
	glog.Info("===============")
	for _, r := range successes {
		glog.Infof("%s %s", r.Repo, r.PullRequest)
	}
	glog.Info("===============")
	glog.Info("⏭  SKIPPED ⏭ ")
	glog.Info("===============")
	for _, r := range skips {
		glog.Infof("%s %s", r.Repo)
	}

	glog.Info("===============")
	glog.Info("🚨 FAILED 🚨")
	glog.Info("===============")
	for _, r := range failures {
		glog.Infof("%s %s", r.Repo)
	}
}

func saveResults(allResults map[string]*runResults) {}

func logResults(r *runResults) {
	switch r.ExitCode {
	case 0:
		glog.Infof("%s: ✅ SUCCESS", r.Repo)
		if r.PullRequest != "" {
			glog.Infof("%s: Pull Request: %s", r.Repo, r.PullRequest)
		}
		glog.Infof("%s: ✅ SUCCESS", r.Repo)
	case skipExitCode:
		glog.Infof("%s: ⏭  SKIPPED", r.Repo)
	default:
		glog.Infof("%s: 🚨 FAILED, exited with %d", r.Repo, r.ExitCode)
		errLines := strings.Split(r.Stderr, "\n")
		glog.Errorf("%s: Error: %s...", r.Repo, errLines[0])
	}
}

type runResults struct {
	Repo        string `json:"repo"`
	Stdout      string `json:"stdout"`
	Stderr      string `json:"stderr"`
	ExitCode    int    `json:"exitCode"`
	PullRequest string `json:"pullRequest"`
}

func runRepo(repoName string) (*runResults, error) {
	repoPath := filepath.Join(workspace, repoName)
	repo, err := checkoutRepo(repoName, repoPath)
	if err != nil {
		return nil, err
	}

	err = checkoutBranch(repo)
	if err != nil {
		return nil, err
	}

	stdout, stderr, exitCode, err := runScriptInRepo(repoPath)
	var prURL string
	if makePr && exitCode == 0 {
		prURL, err = makePullRequest(repo)
		if err != nil {
			return nil, err
		}
	}

	r := &runResults{
		Repo:        repoName,
		ExitCode:    exitCode,
		Stdout:      string(stdout),
		Stderr:      string(stderr),
		PullRequest: prURL,
	}
	return r, nil
}

func makePullRequest(repo *git.Repository) (string, error) {
	wt, err := repo.Worktree()
	if err != nil {
		return "", err
	}

	_, err = wt.Add(".")
	if err != nil {
		return "", err
	}

	glog.Info("📝 Making Pull Request")
	prCmd := exec.Command("gh", "pr", "create", "-t", title, "-b", description)
	stdout, err := prCmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	prURL, err := ioutil.ReadAll(stdout)
	if err != nil {
		return "", err
	}

	return string(prURL), nil
}

func runScriptInRepo(repoPath string) (stdoutBytes []byte, stderrBytes []byte, exitCode int, err error) {
	scriptCmd := exec.Command(script)
	scriptCmd.Dir = repoPath
	stdout, _ := scriptCmd.StdoutPipe()
	stderr, _ := scriptCmd.StderrPipe()

	// Run synchronously, can probably switch to async later
	err = scriptCmd.Run()
	if err != nil {
		return nil, nil, 0, err
	}

	stdoutBytes, err = ioutil.ReadAll(stdout)
	if err != nil {
		return nil, nil, 0, err
	}
	stderrBytes, err = ioutil.ReadAll(stderr)
	if err != nil {
		return nil, nil, 0, err
	}
	return stdoutBytes, stderrBytes, scriptCmd.ProcessState.ExitCode(), nil
}

func checkoutBranch(repo *git.Repository) error {
	head, err := repo.Head()
	if err != nil {
		return err
	}

	wt, err := repo.Worktree()
	if err != nil {
		return err
	}

	checkoutOpts := &git.CheckoutOptions{
		Hash:   head.Hash(),
		Branch: plumbing.ReferenceName(branchName),
		Create: true,
		Force:  true,
		Keep:   false,
	}
	err = wt.Checkout(checkoutOpts)
	if err != nil {
		return err
	}
	return nil
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

func checkoutRepo(repoName string, repoPath string) (repo *git.Repository, err error) {
	if isDir(repoPath) {
		return git.PlainOpen(repoPath)
	} else {
		return cloneRepo(repoName, repoPath)
	}
}

func cloneRepo(repoName string, dest string) (*git.Repository, error) {
	glog.Info("%s: 🧘‍♂️ Cloning (this could take a while...)", repoName)
	githubRepoURL := fmt.Sprintf("git@github.com:%s/%s", org, repoName)

	// Should probably set some clone options here for convenience and safety
	// TODO: set clone depth = 1
	repo, err := git.PlainClone(githubRepoURL, false, nil)
	if err != nil {
		return nil, fmt.Errorf("%s: Error cloning repository: %s", repoName, err.Error())
	}
	return repo, nil
}
