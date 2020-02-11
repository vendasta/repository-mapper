# Repository Mapper

<!-- toc -->

- [Introduction](#introduction)
- [Arguments](#arguments)
  * [Named Arguments](#named-arguments)
  * [Positional Arguments](#positional-arguments)
- [Script](#script)
- [Example](#example)
- [Using All Repositories](#using-all-repositories)
- [README Table of Contents Generation](#readme-table-of-contents-generation)

<!-- tocstop -->

## Introduction

**NOTE**: Repository Mapper requires Bash version >= 4; you'll probably need to `brew install bash`.

Repository Mapper can do a lot of work for you, if you provide a script it will run it on every repository you want.

It can help with things like:

* Running structured queries on every repository, E.g.
  - Which repositories still use 'go dep'?
  - How many repositories still depend on X version of this package?
  - Find me all usages of the term 'X' across all repositories
  - Which users contribute to which repositories?

* Running scripts and creating pull requests on every repository. E.g.
  - Auto upgrade X dependency in every repository
  - Add this LICENCE file to every repository

Let's look at an example invocation to break it down.

```bash
ORG=vendasta \
 NO_FETCH=true \
 BRANCH_NAME=repository-mapper/contributors \
 SCRIPT="$(realpath ./scripts/get-contributors.sh)" \
 ./repository-mapper.sh repo1 repo2 repo3
```

Let's break it down.

## Arguments

Here are the currently available options:

### Named Arguments

Set these as environment variables when running the script:

* `ORG` (required): The Github Organization where your repositories are stored.
* `SCRIPT` (required): A bash invocation to run. See [script](#script) below for notes on how this should work.
* `BRANCH_NAME` (required): The working branch for the mapping process. Any repository with an existing branch of this name will have that branch clobbered locally. However, if there is an already-existing branch of that same name on the remote and `MAKE_PR=yes` is set, it will not force push, nor will it make a PR.
* `NO_FETCH=true` (optional): Specify not to re-fetch latest master on all repos. This can speed up your script substantially, but means the working copy of each repo may be out of date.
* `MAKE_PR` (optional): Whether repository mapper should commit, push, and make a PR to the provided branch on Github. Set to `MAKE_PR=yes` to do so.
* `PR_TITLE` (required if MAKE_PR=yes): The PR title
* `PR_DESCRIPTION` (required if MAKE_PR=yes): The PR description

### Positional Arguments

* Repos: All positional arguments are repository names to run the script on. Simply provide the short-form name of the repo; e.g. 'my-repo' or 'another-repo'. The organization name will automatically be appended.
* To use all recently updated repositories in the organization, see [using all repositories](#all-repositories).

## Script

The provided script will be run using `bash -c "$SCRIPT"` from within the root of each directory. This means that all references to files should use absolute paths.

All stdout, stderr, and exit code will automatically be collected for you.

If a script returns a non-zero exit code, repository mapper will not create a commit or pull request in that repository.

You can exit a script with exit code `10` to "skip" the repository and signify there's no work to be done.

Optionally, if you want to store more structured results you can write a **single** JSON results object to file descriptor 3 within your script and it will be added to the collected results.

For example, the following script collects all the top-level files into an array which will be stored in the results of the script. See the [`jq` manual](https://stedolan.github.io/jq/manual/v1.6/) for how to work with JSON in bash.

```bash
#!/bin/bash
echo "Hello; let me get those files for you!"
jq -n --arg files "$(ls)" '$files | split("\n")' >&3

echo "This is what an error looks like" >&2

exit 42
```

This will result in the result object:

```json
{
    "repo": "my-repo",
    "stdout": "Hello; let me get those files for you!",
    "stderr": "This is what an error looks like",
    "exit_code": 42,
    "pull_request": "",
    "result": ["file-1.txt", "file-2.txt"]
  }
```

Here's a script which gets the list of contributors from each repo as a JSON array. You can find this inside `./scripts/get-contributors.sh`

```bash
#!/bin/bash
contributors=$(git shortlog --summary --numbered --email | cut -f2)
numContributors=$(echo "$contributors" | wc -l)

jq -n --argjson "numContributors" "$numContributors" --arg "contributors" "$contributors" '{numContributors: $numContributors, contributors: $contributors | split("\n")}' >&3
```

## Example

Example command

```bash
$ ORG=vendasta \
   NO_FETCH=true \
   BRANCH_NAME=repository-mapper/contributors \
   SCRIPT="$(realpath ./scripts/get-contributors.sh)" \
   ./repository-mapper.sh repo1 repo2

[1/2] repo1: 🦴🐕 Fetching latest master
[1/2] repo1: 🏃‍♀️ Running script
[1/2] repo1: ✅ SUCCESS

[2/2] repo2: 🦴🐕 Fetching latest master
[2/2] repo2: 🏃‍♀️ Running script
[2/2] repo2: ✅ SUCCESS

===============
✅ SUCCEEDED ✅
===============
repo1
repo2

=============
⏭  SKIPPED ⏭
=============

============
🚨 FAILED 🚨
============

Job results (and stdout/stderr transcripts) available in ./results/repository-mapper/contributors.json
```

And what the results file looks like

```json
[
  {
    "repo": "repo1",
    "stdout": "",
    "stderr": "",
    "exit_code": 0,
    "pull_request": "",
    "result": {
      "numContributors": 3,
      "contributors": [
        "Contributor 1 <contributor1@example.com>",
        "Contributor 2 <contributor2@example.com>",
        "Contributor 3 <contributor3@example.com>"
      ]
    }
  },
  {
    "repo": "repo2",
    "stdout": "",
    "stderr": "",
    "exit_code": 0,
    "pull_request": "",
    "result": {
      "numContributors": 4,
      "contributors": [
        "Contributor 1 <contributor1@example.com>",
        "Contributor 2 <contributor2@example.com>",
        "Contributor 3 <contributor3@example.com>",
        "Contributor 4 <contributor3@example.com>"
      ]
    }
  }
]
```

## Using All Repositories

If you need to simply get an up-to-date list of all active repositories in your org you can run the `get-all-repos` script in the scripts directory. 
It lists to stdout every repo in your org edited in the last year.
