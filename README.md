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

* Running structured queries on every repository
  - E.g. "Which repositories still use 'go dep'?"
  - E.g. "How many repositories still depend on X version of this package?"
  - E.g. "Find me all usages of the term "X" across all repositories

* Running scripts and creating pull requests on every repository
  - E.g. Auto upgrade X dependency in every repository
  - E.g. Add this LICENCE file to every repository

Let's look at an example invocation to break it down.

```bash
MAKE_PR=yes \
  PR_TITLE="Upgrade SDK" \
  PR_DESCRIPTION="This is a critical upgrade" \
  BRANCH_NAME="repository-mapper/upgrade-sdk" \
  SCRIPT="./upgrade-sdk.sh" \
  ./repository-mapper.sh \
  repo1 repo2 repo3
```

Let's break it down.

## Arguments

Here are the currently available options:

### Named Arguments

Set these as environment variables when running the script:

* `SCRIPT` (required): A bash invocation to run. See [script](#script) below for notes on how this should work.
* `BRANCH_NAME` (required): The branch to commit and push to; ENSURE this is UNIQUE across ALL REPOS. (don't worry it won't force-push, but it will fail to make the PR)
* `NO_FETCH=true` (optional): Specify not to re-fetch latest master on all repos. This can speed up your script substantially, but may make PRs against an out-of-date master branch.
* `MAKE_PR` (optional): Whether repository mapper should commit, push, and make a PR to the provided branch on Github. Set to `MAKE_PR=yes` to do so.
* `PR_TITLE` (required if MAKE_PR=yes): The PR title
* `PR_DESCRIPTION` (required if MAKE_PR=yes): The PR description

### Positional Arguments

* Repos: All positional arguments are repository names to run the script on. Simply provide the short-form name of the repo; e.g. 'my-repo' or 'another-repo'
* To use all recently updated repositories in the organization, see [using all repositories](#all-repositories).

## Script

The provided script will be run as a bash script with `bash -c`

All stdout, stderr, and exit code will automatically be collected for you.

If a script returns exit code `10` it will 'skip' the repository, meaning it will not create a commit or pull request in that repository.

Optionally, if you want to store more structured results you can write a **single** JSON results object to file descriptor 3 within your script and it will be added to the collected results.

For example, the following script collects all the top-level files into an array which will be stored in the results of the script. See the [`jq` manual](https://stedolan.github.io/jq/manual/v1.6/) for how to work with JSON in bash.

```bash
#!/bin/bash
echo "Hello; let me get those files for you!"
jq -n --arg files "$(ls)" '$files | split("\n")' >&3

echo "This is what an error looks like" >&2

exit 42
```

Here's a script which gets the list of contributors from each repo as a JSON array.

```bash
#!/bin/bash
contributors=$(git shortlog --summary --numbered --email | cut -f2)
numContributors=$(echo contributors | wc -l)

jq -n --argjson "numContributors" "$numContributors" --arg "contributors" "$contributors" '{numContributors: $numContributors, contributors: $contributors | split("\n")}' >&3
```

## Example

Example command

```bash
$ BRANCH_NAME=repository-mapper/contributors SCRIPT="$(realpath ./scripts/get-contributors.sh)" ./repository-mapper.sh repo1 repo2
repo1: 🦴🐕 Fetching latest master
repo1: 🏃‍♀️ Running script
repo1: ✅ SUCCESS

repo2: 🦴🐕 Fetching latest master
repo2: 🏃‍♀️ Running script
repo2: ✅ SUCCESS

===============
✅ SUCCEEDED ✅
===============
notifications
email

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

## README Table of Contents Generation

This repository uses [markdown-toc](https://github.com/jonschlinkert/markdown-toc) to generate its table of contents. Simply run `npm run regen-toc` to regenerate the table of contents in this file.
