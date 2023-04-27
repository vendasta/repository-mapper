## 0.5.0

- Changed:
    - You can now specify an optional default branch
    - Revert go-git downgrade so .gitignore files are respected
        - If a repo you are running repository-mapper on exists in the `$HOME/repository-mapper` directory it will be deleted
          so your script can run successfully rather than erring out

## 0.4.0

- Changed:
    - If a repository is clean do not open an empty PR when `--make-pr` flag is passed
- Fixed:
    - When a repository exists in the directory repository-mapper stores repositories in `fetch` would error. This is
      a [bug with go-git](https://github.com/go-git/go-git/issues/328). Downgrading to 5.3.0 seems to fix it.

## 0.3.0

- Changed:
    - Updated dependencies

## 0.2.0

- Changed:
    - Make GitHub token required when `--make-pr` flag is specified, would error out otherwise
    - Updated dependencies
- Fixed:
    - When creating a results directory repository mapper stores the results in a file named after provided branch. If
      the branch name contained a forward slash it would treat it as part of the filepath and error due to missing
      directory. Convert forward slash to `-` for results file name to prevent error.

## 0.1.0

- Changed:
    - Exit on failure in sub-shells
