#!/usr/local/bin/bash

# Exit on any errors
set -e
# set -x

SKIP_EXIT_CODE=10
WORKSPACE="$HOME/repository-mapper"
ORG="${ORG:-vendasta}"

if [[ $# -lt 1 ]]; then
    echo 'example usage:' >&2
    cat <<EOF
    MAKE_PR=yes \
      PR_TITLE="Upgrade Dependencies" \
      PR_DESCRIPTION="Upgrades a golang dependency" \
      BRANCH_NAME="repository-mapper/upgrade-golang-deps" \
      SCRIPT=\"\$(realpath ./upgrade-deps.sh) github.com/vendasta/a-golang-dep\" \
      ./repository-mapper.sh \
      repo1 repo2
EOF
    exit 1
fi

if [[ -z "$ORG" ]]; then
    echo 'Choose a github organization with ORG="my-org"' >&2
    exit 1
fi

if [[ -z "$SCRIPT" ]]; then
    echo "Choose a bash invocation to run with SCRIPT=\"\$(realpath ./my-script.sh) arg1 arg2\"" >&2
    echo "Whatever you pass will be run with 'bash -c'" >&2
    echo "Ensure that the script is EXECUTABLE (chmod +x my-file.sh)" >&2
    echo "All scripts must be passed with an ABSOLUTE path, e.g. SCRIPT=\"\$(realpath ./my-script.sh) arg1 arg2\"" >&2
    exit 1
fi

if [[ -z "$BRANCH_NAME" ]]; then
  echo "Choose a branch name with BRANCH_NAME='my-branch-name' ..." >&2
  echo "It should be globally unique across all repos" >&2
  exit 1
fi

if [[ "$MAKE_PR" = "yes" ]]; then
   pull_requests=$(mktemp)
   if ! command -v gh >/dev/null 2>&1; then
      echo "github cli is required to make PR's" >&2
      echo "brew install github/gh/gh" >&2
      echo "Ensure you've authorized it to make pull requests" >&2
      echo "try running 'gh issue list' to see if it succeeds" >&2
      exit 1
   fi

  if [[ -z "$PR_TITLE" ]]; then
    echo "Choose a PR title with PR_TITLE='My PR Title' ..." >&2
    exit 1
  fi
  if [[ -z "$PR_DESCRIPTION" ]]; then
    echo "Choose a PR description with PR_DESCRIPTION='My PR Description' ..." >&2
    exit 1
  fi
fi

if [[ "$@" ]]; then
    # Use arguments as repos
    repos=("$@")
else
    echo "Specify which repos you want to run on" >&2
    exit 1
fi

# Bash associative arrays where we'll keep job results for each repo
declare -A outputs
declare -A errors
declare -A jsons
declare -A pull_requests
declare -A exit_codes

checkout_repo() {
    repo="$1"
    repoPath="$WORKSPACE/$repo"
    # Clone if missing
    if ! [[ -d "$repoPath" ]]; then
       echo "$repo: 🧘‍♂️ Cloning (this could take a while...)"
       git clone "git@github.com:$ORG/$repo" "$repoPath"
    fi

    # create new branch from latest master
    (
      cd "$repoPath"
      if [[ -z $NO_FETCH ]]; then  
          echo "$repo: 🦴🐕 Fetching latest master"
          git fetch origin master >/dev/null 2>&1
      fi
      git checkout "origin/master" -f -B "$BRANCH_NAME" >/dev/null 2>&1
      git reset --hard "origin/master" >/dev/null 2>&1
    )
}

run_script_in_repo() {
    repo="$1"
    idx="$2"
    total_repos="$3"
    repoPath="$WORKSPACE/$repo"

    # Files to stash script output into
    err_file=$(mktemp)
    out_file=$(mktemp)
    json_file=$(mktemp)
    pull_request_file=$(mktemp)
    exit_code="0"

    echo "[$idx/$total_repos] $repo: 🏃‍♀️ Running script (script output is being captured)"
    # Run the script in a subshell and collect stdout, stderr, and any json result
    ( cd "$repoPath"
      # Run provided script
      bash -c "$SCRIPT"
    ) > "$out_file" 2> "$err_file" 3> "$json_file" || exit_code="$?"

    if [[ "$MAKE_PR" = "yes" ]] && [[ $exit_code -eq 0 ]]; then
        (
        cd "$repoPath"
            git add -A
            git commit -m "$PR_TITLE" -m "$PR_DESCRIPTION"
            # Push branch
            git push -u origin HEAD || exit_code="$?"
            if [[ $exit_code -eq 1 ]]; then
                echo "Git push failed, are you sure your branch name doesn't exist on the remote?" >&2
                exit 1
            fi
            # Make pull request
            echo "$repo: 📝 Making Pull Request"
            gh pr create -t "🤖 $PR_TITLE" -b "$PR_DESCRIPTION" >> "$pull_request_file"
        ) || true # maybe there were no changed files; we'll just print errors and keep moving.
    fi

    json_result=$(cat "$json_file")
    # If we didn't get a result, use empty object
    if ! [[ "$json_result" ]]; then
        json_result="{}"
    fi
    # Check if JSON is valid
    if ! echo "$json_result" | jq type > /dev/null; then
        json_result="{}"
        jsonErr="Couldn't parse result: $json_result"
        echo "$jsonErr" >> "$err_file"
        echo "$jsonErr" >&2
    fi
    jsons[$repo]="$json_result"
    exit_codes[$repo]="$exit_code"
    outputs[$repo]="$(cat "$out_file")"
    errors[$repo]="$(cat "$err_file")"
    pull_requests[$repo]="$(cat "$pull_request_file")"

    case "$exit_code" in
        0)
            echo "$repo: ✅ SUCCESS"
            ;;
        $SKIP_EXIT_CODE)
            echo "$repo: ⏭  SKIPPED"
            ;;
        *)
            echo "$repo: 🚨 FAILED, exited with $exit_code"
            ;;
    esac
}

map_repos() {
    num_repos=${#repos[@]}
    for i in "${!repos[@]}"; do
        repo=${repos[i]}
        if ! checkout_repo "$repo" >&2; then
            echo "Error checking out $repo" >&2
            errors[$repo]+="Error checking out $repo; can't run script"
            exit_codes[$repo]=1
            continue
        fi

        run_script_in_repo "$repo" "$(($i + 1))" "$num_repos"
    done
}

collect_results() {
  results='[]'

  # Merge all migration results together
  for repo in ${repos[*]} ; do
      results="$(
          echo "$results" | \
          jq \
            --arg "repo" "$repo" \
            --arg "stdout" "${outputs[$repo]}" \
            --arg "stderr" "${errors[$repo]}" \
            --arg "pull_request" "${pull_requests[$repo]}" \
            --argjson "result" "${jsons[$repo]:-{\}}" \
            --argjson "exit_code" "${exit_codes[$repo]:-1}" \
            '. + [ {repo: $repo, stdout: $stdout, stderr: $stderr, exit_code: $exit_code, pull_request: $pull_request, result: $result} ]')"
  done
  
  results_file="./results/${BRANCH_NAME}.json"
  mkdir -p "./results/$(dirname "$BRANCH_NAME")"
  echo "$results" > "$results_file"
  
  echo ""
  echo "==============="
  echo "✅ SUCCEEDED ✅"
  echo "==============="
  jq -r '.[] | select(.exit_code == 0) | .repo' < "$results_file" 
  
  echo ""
  echo "============="
  echo "⏭  SKIPPED ⏭ "
  echo "============="
  jq -r ".[] | select(.exit_code == $SKIP_EXIT_CODE) | .repo" < "$results_file" 
  
  echo ""
  echo "============"
  echo "🚨 FAILED 🚨"
  echo "============"
  jq -r ".[] | select(.exit_code != 0 and .exit_code != $SKIP_EXIT_CODE) | .repo" < "$results_file" 
  
  
  if [[ "$MAKE_PR" = "yes" ]]; then
      echo ""
      echo "==================="
      echo "📝 Pull Requests 📝"
      echo "==================="
      jq -r '.[] | select(.pull_request and .pull_request != "") | "\(.repo): \(.pull_request)"' < "$results_file" 
  fi

  echo ""
  echo "Job results (and stdout/stderr transcripts) available in ./results/${BRANCH_NAME}.json"
}

# Collect results no matter how we exit
trap collect_results EXIT

# Do the work
map_repos
