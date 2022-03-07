#!/bin/bash
SDK_VERSION=$(cat go.mod | grep "github.com/vendasta/gosdks/logging" | awk '{print $2}')

echo "Logging SDK version on master: ${SDK_VERSION}"
sed -i '' 's|.*github.com/vendasta/gosdks/logging.*|\tgithub.com/vendasta/gosdks/logging v1.13.0|' go.mod
echo "Logging SDK version for PR   : ${SDK_VERSION}"

go mod vendor
