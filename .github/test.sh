#!/bin/bash

set -e

# setup env
export GO111MODULE=on

module_name=$(cat go.mod | grep module | cut -d ' ' -f 2-2)
module_list=(`go list ./...`)
echo "module_name is $module_name"

echo 'mode: atomic' > coverage.txt

for ele in "${module_list[@]}";
do
  echo "start handle sub_module: $ele"
  go test -covermode=atomic -coverprofile=coverage.tmp -coverpkg=./... -parallel 1 -p 1 -count=1 -gcflags=-l $ele | rerun-fail-test -retry-times=5 -- -covermode=atomic -coverprofile=coverage.tmp -coverpkg=./... -parallel 1 -p 1 -count=1 -gcflags=-l $ele
  tail -n +2 coverage.tmp >> coverage.txt || echo ""
  rm coverage.tmp || echo ""
done

# go tool cover -html=coverage.txt
