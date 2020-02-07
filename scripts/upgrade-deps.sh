#!/bin/bash

if [[ -z "$1" ]]; then
  echo "USAGE: upgrade-deps.sh github.com/vendasta/golang-dep github.com/vendasta/golang-dep/sdks/go" >&2
  exit 1
fi

deps="$@"
found_any_deps=""

go_mod_files=$(find . -type f -not -path "*/vendor/*" -iname go.mod)
go_mod_dirs=$(echo "$go_mod_files" | xargs -n 1 dirname)

upgrade_deps() {
  found_this_dep=""
  # only upgrade if the module has the dependency
  for dep in $deps ; do
    if grep -e "\s$dep\s" "go.mod" ; then
      found_this_dep="yes"
      go get "$dep"
    fi
  done

  if [[ "$found_this_dep" = "yes" ]]; then
    go mod tidy
    go mod vendor
    return 20
  fi
}

for dir in $go_mod_dirs ; do
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
