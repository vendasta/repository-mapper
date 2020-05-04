#!/bin/bash

set -e

go_mod_files=$(find . -type f -not -path "*/vendor/*" -iname go.mod)

if [[ -z "$go_mod_files" ]]; then
  exit 10
fi

for f in $go_mod_files ; do
  (
    if grep 'github.com/vendasta/thor' $f -q; then
      echo "$f"
    fi
  )
done
exit 10
