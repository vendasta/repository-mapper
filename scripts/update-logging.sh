#!/bin/bash
MIN='v1.13.3'

if [ ! -f go.mod ]; then
  echo "No go.mod file present, skipping"
  exit 10
fi
SDK_VERSION=$(cat go.mod | grep "github.com/vendasta/gosdks/logging" | awk '{print $2}')

if [[ -z $SDK_VERSION ]]; then
  echo "logging sdk not present, skipping"
  exit 10
fi

SDK_PIECES=(${SDK_VERSION//./ })

echo "Logging SDK version on master: ${SDK_VERSION}"
if [[ ${SDK_PIECES[1]} -lt 13 ]]; then
  go get github.com/vendasta/gosdks/logging
  go mod vendor
elif [[ ${SDK_PIECES[1]} -eq 13 ]]; then
  if [[ ${SDK_PIECES[2]} -lt 3 ]]; then
    go get github.com/vendasta/gosdks/logging
    go mod vendor
  else
    echo "Logging version high enough, skipping"
    exit 10
  fi
else
  echo "Logging version high enough, skipping"
  exit 10
fi
