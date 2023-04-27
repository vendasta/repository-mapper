package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"time"

	"github.com/go-git/go-git/v5"
	gitconfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	gitobject "github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	git_ssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

func initAuth() (err error) {
	if userName != "" && authToken != "" {
		auth = &http.BasicAuth{
			Username: userName,
			Password: authToken,
		}
	} else {
		auth, err = git_ssh.NewPublicKeysFromFile("git", rsaKeyFile, rsaKeyPassword)
		if err != nil {
			return err
		}

	}

	return nil
}

func checkoutRepo(repoName, repoPath, defaultBranch string) (repo *git.Repository, err error) {
	fmt.Printf("%s: Checking out at %s\n", repoName, repoPath)
	if isDir(repoPath) {
		fmt.Printf("%s: Repository exists\n", repoName)
		repo, err = git.PlainOpen(repoPath)
		if err != nil {
			return nil, err
		}
		if !noFetch {
			wt, err := repo.Worktree()
			if err != nil {
				fmt.Printf("Failed to get repo handle: %s\n", err)
				return nil, err
			}
			err = wt.Checkout(&git.CheckoutOptions{
				Branch: plumbing.ReferenceName(defaultBranch),
				Force:  true,
			})
			if err != nil {
				fmt.Printf("Error checking out %s: %s\n", defaultBranch, err)
				return nil, err
			}
			fmt.Printf("%s: Fetching latest %s (could take a minute) ⏱\n", defaultBranch, repoName)
			opts := &git.FetchOptions{
				RemoteName: "origin",
				Depth:      1,
				Auth:       auth,
				// Fetch only latest default branch
				RefSpecs: []gitconfig.RefSpec{gitconfig.RefSpec(fmt.Sprintf("+refs/heads/%s:refs/remotes/origin/%s", defaultBranch, defaultBranch))},
			}
			err = repo.Fetch(opts)
			if err != nil && err != git.NoErrAlreadyUpToDate {
				return nil, fmt.Errorf("error fetching: %w", err)
			}
		}
		return repo, nil
	} else {
		return cloneRepo(repoName, repoPath, defaultBranch)
	}
}

func cloneRepo(repoName, dest, defaultBranch string) (*git.Repository, error) {
	fmt.Printf("%s: 🧘‍♂️ Cloning (this could take a while...)\n", repoName)
	githubRepoURL := fmt.Sprintf("https://github.com/%s/%s", org, repoName)
	cloneOptions := &git.CloneOptions{
		URL:           githubRepoURL,
		ReferenceName: plumbing.NewBranchReferenceName(defaultBranch),
		SingleBranch:  true,
		Depth:         1,
		Auth:          auth,
	}
	repo, err := git.PlainClone(dest, false, cloneOptions)
	if err != nil {
		return nil, fmt.Errorf("rror cloning repository: %w", err)
	}
	return repo, nil
}

func checkoutBranch(repoName string, repo *git.Repository, defaultBranch string) error {
	wt, err := repo.Worktree()
	if err != nil {
		return err
	}

	masterRef, err := repo.Reference(plumbing.NewBranchReferenceName(defaultBranch), true)
	if err != nil {
		return fmt.Errorf("error getting reference: %w", err)
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
		fmt.Printf("%s: Resetting branch to latest %s\n", defaultBranch, repoName)
		resetOpts := &git.ResetOptions{
			Commit: masterRef.Hash(),
			Mode:   git.HardReset,
		}
		err = wt.Reset(resetOpts)
		if err != nil {
			return fmt.Errorf("error resetting branch: %w", err)
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

// Make a pull request
func makePullRequest(repoName string, repoPath string, repo *git.Repository) (string, error) {
	// sleep to make sure file system/git actually picks up changes
	time.Sleep(200 * time.Millisecond)
	wt, err := repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("error getting worktree: %w", err)
	}

	st, err := wt.Status()
	if err != nil {
		return "", fmt.Errorf("error checking git status: %w", err)
	}
	if st.IsClean() {
		return "", nil
	}
	// Add all changed files
	_, err = wt.Add(".")
	if err != nil {
		return "", fmt.Errorf("error adding changes: %w", err)
	}

	commiter := &gitobject.Signature{
		Name:  gitAuthor,
		Email: gitAuthorEmail,
		When:  time.Now().UTC(),
	}
	// Commit changes
	commitOpts := &git.CommitOptions{
		Author:    commiter,
		Committer: commiter,
	}
	fmt.Printf("%s: 📝 Committing Changes\n", repoName)
	_, err = wt.Commit(title, commitOpts)
	if err != nil {
		return "", fmt.Errorf("error committing changes: %w", err)
	}

	// Push to origin
	pushOpts := &git.PushOptions{
		RemoteName: "origin",
		Auth:       auth,
	}
	fmt.Printf("%s: Setting upstream origin to %s\n", repoName, branchName)
	err = repo.Push(pushOpts)
	if err != nil {
		return "", fmt.Errorf("error during push: %w", err)
	}

	//create pull request
	// TODO: replace gh's command usage with  https://github.com/cli/go-gh
	fmt.Printf("%s: 📝 Making Pull Request\n", repoName)
	prCmd := exec.Command("gh", "pr", "create", "-t", "🤖 "+title, "-b", description, "-H", branchName)
	prCmd.Dir = repoPath

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	prCmd.Stdout = stdout
	prCmd.Stderr = stderr

	cmdErr := prCmd.Run()
	stdoutBytes, err := io.ReadAll(stdout)
	if err != nil {
		return "", err
	}
	stderrBytes, err := io.ReadAll(stderr)
	if err != nil {
		return "", err
	}

	if cmdErr != nil {
		return "", fmt.Errorf("Error creating PR. \nEnsure you've tested gh in a separate terminal first, and then resolve the following errors: \n%s\n%s", cmdErr.Error(), string(stderrBytes))
	}

	prLinkBytes := linkRegex.Find(stdoutBytes)
	return string(prLinkBytes), nil
}
