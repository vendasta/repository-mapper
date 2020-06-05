package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"

	git "github.com/go-git/go-git/v5"
	gitconfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"

	git_ssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"

	"github.com/spf13/cobra"
)

var (
	// cli flags
	branchName     string
	org            string
	script         string
	makePr         bool
	title          string
	description    string
	noFetch        bool
	workspace      string
	auth           transport.AuthMethod
	rsaKeyFile     string
	rsaKeyPassword string

	// constants
	skipExitCode = 10
	homeDir      string
)

// Initialize cobra cli flags and args
func init() {
	// Get the current system user so we can find their home dir
	usr, _ := user.Current()
	homeDir = usr.HomeDir
	workspace = filepath.Join(homeDir, "repository-mapper")

	rootCmd.Flags().StringVarP(&branchName, "branch-name", "b", "", "The branch to create. Should be globally unique.")
	rootCmd.MarkFlagRequired("branch-name")

	rootCmd.Flags().StringVarP(&org, "org", "o", "vendasta", "The github organization the repos live in.")

	rootCmd.Flags().StringVarP(&script, "script", "s", "", "Path to the script to run in each repository")
	rootCmd.MarkFlagRequired("script")

	rootCmd.Flags().BoolVarP(&makePr, "make-pr", "p", false, "Create a PR in each repo after running the script")
	rootCmd.Flags().StringVarP(&title, "title", "t", "", "Title of the PR")
	rootCmd.Flags().StringVarP(&description, "description", "d", "", "Description of the PR")

	defaultRSAKeyFile := filepath.Join(homeDir, ".ssh", "id_rsa")
	rootCmd.Flags().StringVar(&rsaKeyFile, "rsa-key-file", defaultRSAKeyFile, "(optional) The location of an rsa key with github permissions")
	rootCmd.Flags().StringVar(&rsaKeyPassword, "rsa-key-password", "", "(optional) The password for your ssh key if you have one configured")
}

var rootCmd = &cobra.Command{
	Use:          "repository-mapper",
	Short:        "Run scripts on repositories across your org",
	Long:         `Run scripts and queries on repositories across your org`,
	Args:         cobra.MinimumNArgs(1),
	RunE:         run,
	SilenceUsage: true,
}

// The main command logic
func run(cmd *cobra.Command, args []string) error {
	err := validateArgs()
	if err != nil {
		return err
	}

	fmt.Printf("Using script: %s\n", script)

	allResults := map[string]*runResults{}

	// Run each repo in serial
	// Could pretty easily allow running in parallel if we wanted to
	for _, repoName := range args {
		// Defer to the per-repo operations (i.e. cloning, git-ops, running script)
		results, err := runRepo(repoName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %s\n", repoName, err.Error())
			continue
		}
		// Print out the results for this repo
		logResults(results)
		// Stash results for summary
		allResults[repoName] = results
	}

	// Print out summary of all repo results
	summarizeResults(allResults)

	// Save detailed result print out to disk
	err = saveResults(allResults)
	if err != nil {
		return fmt.Errorf("Error saving results: %s\n", err.Error())
	}
	return nil
}

// Print all the results to console
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

	fmt.Println("\n===============")
	fmt.Println("✅ SUCCEEDED ✅")
	fmt.Println("===============")
	for _, r := range successes {
		fmt.Printf("%s %s\n", r.Repo, r.PullRequest)
	}
	fmt.Println("\n===============")
	fmt.Println("⏭  SKIPPED ⏭ ")
	fmt.Println("===============")
	for _, r := range skips {
		fmt.Println(r.Repo)
	}

	fmt.Println("\n===============")
	fmt.Println("🚨 FAILED 🚨")
	fmt.Println("===============")
	for _, r := range failures {
		fmt.Println(r.Repo)
	}
	// spacer
	fmt.Println("")
}

func saveResults(allResults map[string]*runResults) error {
	// Ensure results dir exists
	os.MkdirAll("./results", os.ModePerm)
	fp := filepath.Join(".", "results", branchName+".json")
	data, err := json.Marshal(allResults)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(fp, data, os.ModePerm)
	if err != nil {
		return err
	}
	fmt.Printf("Results written to ./%s\n", fp)
	return nil
}

// Log results from a single repo run
func logResults(r *runResults) {
	switch r.ExitCode {
	case 0:
		fmt.Printf("%s: ✅ SUCCESS\n", r.Repo)
		if r.PullRequest != "" {
			fmt.Printf("%s: Pull Request: %s\n", r.Repo, r.PullRequest)
		}
	case skipExitCode:
		fmt.Printf("%s: ⏭  SKIPPED\n", r.Repo)
	default:
		fmt.Printf("%s: 🚨 FAILED, exited with %d\n", r.Repo, r.ExitCode)
		errLines := strings.Split(r.Stderr, "\n")
		if errLines[0] != "" {
			fmt.Fprintf(os.Stderr, "%s: Error: %s...\n", r.Repo, errLines[0])
		}
	}
}

// Results from a single repo run
type runResults struct {
	Repo        string `json:"repo"`
	Stdout      string `json:"stdout"`
	Stderr      string `json:"stderr"`
	ExitCode    int    `json:"exitCode"`
	PullRequest string `json:"pullRequest"`
}

// Perform all necessary tasks for a single repo
func runRepo(repoName string) (*runResults, error) {
	repoPath := filepath.Join(workspace, repoName)
	repo, err := checkoutRepo(repoName, repoPath)
	if err != nil {
		return nil, err
	}

	// Checkout the desired branch name tracking from latest master
	err = checkoutBranch(repoName, repo)
	if err != nil {
		return nil, err
	}

	// Run the script inside the repo
	stdout, stderr, exitCode, err := runScriptInRepo(repoName, repoPath)
	if err != nil {
		return nil, err
	}

	var prURL string
	// Only make a PR if the script succeeded and the flag is set
	if makePr && exitCode == 0 {
		prURL, err = makePullRequest(repoName, repo)
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

// Make a pull request (requires the `gh` cli tool)
func makePullRequest(repoName string, repo *git.Repository) (string, error) {
	wt, err := repo.Worktree()
	if err != nil {
		return "", err
	}

	_, err = wt.Add(".")
	if err != nil {
		return "", err
	}

	fmt.Println("📝 Making Pull Request")
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

func runScriptInRepo(repoName, repoPath string) (stdoutBytes []byte, stderrBytes []byte, exitCode int, err error) {
	scriptCmd := exec.Command(script)
	scriptCmd.Dir = repoPath
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	scriptCmd.Stdout = stdout
	scriptCmd.Stderr = stderr

	fmt.Printf("%s: 🏃‍♂️ Running script\n", repoName)
	// Run synchronously, can probably switch to async later
	err = scriptCmd.Run()
	// err is returned on non-zero script exit codes, so we check specifically for something OTHER than an ExitError
	if _, ok := err.(*exec.ExitError); err != nil && !ok {
		return nil, nil, 0, fmt.Errorf("Error running script: %s", err.Error())
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

func checkoutBranch(repoName string, repo *git.Repository) error {
	wt, err := repo.Worktree()
	if err != nil {
		return err
	}

	masterRef, err := repo.Reference(plumbing.NewBranchReferenceName("master"), true)
	if err != nil {
		return fmt.Errorf("Error getting reference: %s", err.Error())
	}

	_, err = repo.Reference(plumbing.NewBranchReferenceName(branchName), false)
	if err == nil {
		checkoutOpts := &git.CheckoutOptions{
			Branch: plumbing.NewBranchReferenceName(branchName),
			Force:  true,
			Keep:   false,
		}
		err = wt.Checkout(checkoutOpts)
		if err != nil {
			return err
		}
		fmt.Printf("%s: Resetting branch to latest master\n", repoName)
		resetOpts := &git.ResetOptions{
			Commit: masterRef.Hash(),
			Mode:   git.HardReset,
		}
		err = wt.Reset(resetOpts)
		if err != nil {
			return fmt.Errorf("Error resetting branch: %s", err.Error())
		}
		return nil
	}

	checkoutOpts := &git.CheckoutOptions{
		Hash:   masterRef.Hash(),
		Branch: plumbing.NewBranchReferenceName(branchName),
		Create: true,
		Force:  true,
		Keep:   false,
	}

	fmt.Printf("%s: Creating new branch\n", repoName)
	err = wt.Checkout(checkoutOpts)
	if err != nil {
		return err
	}
	return nil
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
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

	err = initAuth()
	if err != nil {
		return err
	}

	if makePr {
		_, err := exec.LookPath("asldkfj")
		if err != nil {
			return fmt.Errorf("The github cli is required to make a pull request. Please run:\nbrew install github/gh/gh")
		}
		if title == "" {
			return fmt.Errorf("A PR title is required")
		}
		if description == "" {
			return fmt.Errorf("A PR description is required")
		}
	}

	return nil
}

func initAuth() (err error) {
	auth, err = git_ssh.NewPublicKeysFromFile("git", rsaKeyFile, rsaKeyPassword)
	if err != nil {
		return err
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
	fmt.Printf("%s: Checking out at %s\n", repoName, repoPath)
	if isDir(repoPath) {
		fmt.Printf("%s: Repository exists\n", repoName)
		repo, err = git.PlainOpen(repoPath)
		if err != nil {
			return nil, err
		}
		if !noFetch {
			fmt.Printf("%s: Fetching latest master (could take a minute) ⏱\n", repoName)
			opts := &git.FetchOptions{
				RemoteName: "origin",
				Depth:      1,
				Auth:       auth,
				// Fetch only latest master
				RefSpecs: []gitconfig.RefSpec{"+refs/heads/master:refs/remotes/origin/master"},
			}
			err = repo.Fetch(opts)
			if err != nil && err != git.NoErrAlreadyUpToDate {
				return nil, fmt.Errorf("Error fetching: %s", err.Error())
			}
		}
		return repo, nil
	} else {
		return cloneRepo(repoName, repoPath)
	}
}

func cloneRepo(repoName string, dest string) (*git.Repository, error) {
	fmt.Printf("%s: 🧘‍♂️ Cloning (this could take a while...)\n", repoName)
	githubRepoURL := fmt.Sprintf("https://github.com/%s/%s", org, repoName)

	cloneOptions := &git.CloneOptions{
		URL:           githubRepoURL,
		ReferenceName: "master",
		SingleBranch:  true,
		Depth:         1,
		Auth:          auth,
	}
	repo, err := git.PlainClone(dest, false, cloneOptions)
	if err != nil {
		return nil, fmt.Errorf("Error cloning repository: %s", err.Error())
	}
	return repo, nil
}
