## 0.4.0

- Changed:
    - If a repository is clean do not open an empty PR when `--make-pr` flag is passed
- Fixed:
    - Downgraded go-git dependency to 5.3.0 to fix issue where running repository-mapper on a repo that is already
      downloaded would error

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