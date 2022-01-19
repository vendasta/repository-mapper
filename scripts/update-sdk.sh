#!/bin/bash

file_name=./go.mod
sdk_name=event-broker

sdk_file_version() {
    grep $sdk_name $file_name | awk '{print $2}'
}

if [[ ! -f ${file_name} ]]
then
  echo "$file_name does not exist in this repo"
  exit 10 # skip repo
else
  if (grep -q "$sdk_name" "$file_name")
  then
    actual=$(sdk_file_version)
    echo "$sdk_name version on master is: $actual"
    go get github.com/vendasta/event-broker/sdks/go@v1.4.0
    go mod vendor
    actual=$(sdk_file_version)
    echo "$sdk_name version on PR is: $actual"

  else
      echo "Repo doesn't use $sdk_name"
  fi
fi



