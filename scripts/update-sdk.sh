#!/bin/bash

file_name=./go.mod
sdk_name=event-broker
sdk_new_version=1.5.0

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
    sed -i '' "s|^.*github.com/vendasta/$sdk_name.*|\tgithub.com/vendasta/$sdk_name/sdks/go v$sdk_new_version|" $file_name
    actual=$(sdk_file_version)
    echo "$sdk_name version on master is: $actual"
    # go mod vendor
  else
      echo "Repo doesn't use $sdk_name"
  fi
fi



