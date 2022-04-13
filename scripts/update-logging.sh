#!/bin/bash
WANTED='v1.14.0'
SDK_VERSION=$(cat go.mod | grep "github.com/vendasta/gosdks/logging" | awk '{print $2}')

echo "Logging SDK version on master: ${SDK_VERSION}"
if [[ SDK_VERSION < WANTED ]]
  then
    sed -i '' 's|.*github.com/vendasta/gosdks/logging.*|\tgithub.com/vendasta/gosdks/logging v1.14.0|' go.mod
    echo "Logging SDK version for PR   : ${SDK_VERSION}"

    go mod vendor
  else
    exit 10
fi
