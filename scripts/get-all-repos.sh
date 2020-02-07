#!/bin/bash

set -e

if [[ -z "$GITHUB_USER" ]]; then
  echo "set GITHUB_USER">&2
  exit 1
fi

if [[ -z "$GITHUB_USER" ]]; then
  echo "set GITHUB_TOKEN, you can create a token here:">&2
  echo "https://github.com/settings/tokens" >&2
  exit 1
fi

repos_dir="repos"
mkdir -p "$repos_dir"
all_repos_info_file="$repos_dir/allrepos.json"

refetch_github_repo_info() {
    repo_files=()
    i=1
    while true; do
        repo_file="$(mktemp)"
        echo "Fetching up to date repo info: page $i" >&2
        curl -u "$GITHUB_USER":"$GITHUB_TOKEN" -s "https://api.github.com/orgs/${ORG:-vendasta}/repos?per_page=100&page=$i" > "$repo_file"
        i=$((i + 1))
        repo_files+=("$repo_file")
        if [ "$(jq 'length' <"$repo_file" )" -lt 100 ]; then
            break
        fi
    done

    all_repos_json=$(jq -s add ${repo_files[*]})
    echo "$all_repos_json" > "$all_repos_info_file"
}

# Find existing repos json file, refresh only if it's a week old
if ! [ -f "$(find "$all_repos_info_file" -mmin -$((60 * 24 * 7)) 2>/dev/null)" ]; then
    refetch_github_repo_info
fi

# Filter for repos updated in the last year
filtered_repos=$(cat "$all_repos_info_file" | jq '[.[] | select((.pushed_at | fromdateiso8601) > ("2019-01-01T00:00:00Z" | fromdateiso8601))]')
repo_urls=$(echo "$filtered_repos" | jq -r '.[] | .name')

# Echo out all filtered repo names for calling script
echo "$repo_urls"
