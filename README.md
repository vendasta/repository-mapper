# Repository Mapper (Open Source)

<!-- toc GFM -->

* [Introduction](#introduction)
* [Installation](#installation)
* [Usage](#usage)
* [Arguments](#arguments)
* [Script](#script)
* [Using All Repositories](#using-all-repositories)

<!-- tocstop -->

## Introduction

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

## Installation

```go
go get github.com/vendasta/repository-mapper
```

## Usage

Let's look at an example invocation to break it down.

```bash
repository-mapper \
  --no-fetch \
  --org=vendasta \
  --branch=mapper/contributors \
  --script=./test.sh \
  repo1 repo2 repo3
```

Let's break it down.

## Arguments

Run `repository-mapper -h` for usage information

```bash
$ repository-mapper -h
Run scripts and queries on repositories across your org

Usage:
  repository-mapper [flags] repos...

Flags:
  -b, --branch-name string        The branch to create. Should be globally unique.
  -d, --description string        Description of the PR
  -h, --help                      help for repository-mapper
  -p, --make-pr                   Create a PR in each repo after running the script
  -o, --org string                The github organization the repos live in. (default "vendasta")
      --rsa-key-file string       (optional) The location of an rsa key with github permissions (default "/Users/cpenner/.ssh/id_rsa")
      --rsa-key-password string   (optional) The password for your ssh key if you have one configured
  -s, --script string             Path to the script to run in each repository
  -t, --title string              Title of the PR
```

Pass as many repositories as you like as positional arguments. Simply provide the short-form name of the repo; e.g. 'my-repo' or 'another-repo'. The organization name will automatically be appended.

To use all recently updated repositories in the organization, see [using all repositories](#all-repositories).

## Script

The provide script can be any executable. It will be run without any arguments at the root of each repository.

All stdout, stderr, and exit code will automatically be collected for you and will be recorded into the json file which is written after each run.

If a script returns a non-zero exit code, repository mapper will not create a commit or pull request in that repository.

You can exit a script with exit code `10` to "skip" the repository and signify there's no work to be done.

Here's one example script:

```bash
#!/bin/bash
echo "Hello; let me get those files for you!"
ls

echo "This is what an error looks like" >&2

exit 42
```

This will result in the result object:

```json
{
    "repo": "my-repo",
    "stdout": "Hello; let me get those files for you!\nfile-1.txt file2.txt",
    "stderr": "This is what an error looks like",
    "exit_code": 42,
    "pull_request": ""
  }
```

## Using All Repositories

If you need to simply get an up-to-date list of all active repositories in your org you can run the `get-all-repos` script in the scripts directory. 
It lists to stdout every repo in your org edited in the last year.
