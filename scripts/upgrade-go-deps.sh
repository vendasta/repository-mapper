#!/bin/bash

# An example script showing how to upgrade dependencies of a Go project

go_executable=$(which go)
if [ -z "$go_executable" ]; then
  echo 1>&2 "Could not find a go executable, stopping"
  exit 1
fi

# Add any dependencies you want here. E.g. deps=('github.com/go-git/go-git/v5' 'github.com/spf13/cobra')
deps=()
if [ -z $deps ]; then
  echo 1>&2 "Error: No dependencies specified in script"
  exit 2
fi
found_any_deps=""

go_mod_files=$(find . -type f -not -path "*/vendor/*" -iname go.mod)
go_mod_dirs=$(echo "$go_mod_files" | xargs -n 1 dirname)

upgrade_deps() {
  found_this_dep=""
  # only upgrade if the module has the dependency
  for dep in $deps; do
    if grep -e "\s$dep\s" "go.mod"; then
      found_this_dep="yes"
      go get "$dep"
    fi
  done

  if [[ "$found_this_dep" = "yes" ]]; then
    if [[ -z $(grep "go 1\.2[[:digit:]]" go.mod) ]]; then
      go mod tidy
    fi
    go mod vendor
    return 20
  fi
}

for dir in $go_mod_dirs; do
  exit_code=0
  (
    cd "$dir"
    upgrade_deps
  ) || exit_code="$?"
  if [[ $exit_code = 20 ]]; then
    found_any_deps=yes
  elif [[ $exit_code -ne 0 ]]; then
    exit $exit_code
  fi
done

if [[ "$found_any_deps" = "yes" ]]; then
  exit 0
else
  # Repository Mapper skip-repo code
  exit 10
fi
