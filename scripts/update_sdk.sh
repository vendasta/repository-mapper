#!/bin/bash
SDK_NAME=event-broker
SDK_VERSION=$(cat ./go.mod | grep $SDK_NAME | awk '{print $2}')

if ${SDK_VERSION}
then
  echo "Repo doesn't use $SDK_NAME"
else
  echo "$SDK_NAME version on master is: ${SDK_VERSION}"
  # sed -i '' 's|.*github.com/vendasta/gosdks/logging.*|\tgithub.com/vendasta/gosdks/logging v1.12.0|' go.mod
  # echo "Logging SDK version for PR   : ${SDK_VERSION}"
  # go mod vendor
fi

