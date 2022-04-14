#!/bin/bash
MIN='v1.13.3'

if [ ! -f go.mod ]; then
    echo "No go.mod file present"
    exit 10
fi
SDK_VERSION=$(cat go.mod | grep "github.com/vendasta/gosdks/logging" | awk '{print $2}')

echo "Logging SDK version on master: ${SDK_VERSION}"
if [[ SDK_VERSION < MIN ]]
  then
    go get github.com/vendasta/gosdks/logging
    go mod vendor
  else
    exit 10
fi
