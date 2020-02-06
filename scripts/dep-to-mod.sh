#!/bin/bash

set -e

repo_remainder="$(dirname "$PWD")"
repo_prefix="github.com/${ORG:-vendasta}"
go_mod_files=$(find . -type f -not -path "*/vendor/*" -iname Gopkg.toml)
go_mod_dirs=$(echo "$go_mod_files" | xargs -n 1 dirname)

init() {
  if [[ "$PWD" =~ /v1$ ]]; then
    mv Gopkg.toml Gopkg.lock ../
    cd ..
  fi
  if ! [[ -f "go.mod" ]]; then
    GO111MODULE=on go mod init "$repo_prefix${PWD#"$repo_remainder"}"
    GO111MODULE=on go mod vendor
  fi
  rm Gopkg.toml Gopkg.lock
}

if [[ -z "$go_mod_dirs" ]]; then
  exit 10
fi

for dir in $go_mod_dirs ; do
  ( cd "$dir" && init )
done
