#!/bin/bash -e

tag=$1

for m in $(find ./ -name 'go.mod'); do
  d=$(dirname $m);
  pushd $d;
  grep -q github.com/micro/go-micro/v2 go.mod && go get github.com/micro/go-micro/v2@$tag
  grep -q github.com/micro/micro/v2 go.mod && go get github.com/micro/micro/v2@$tag
  go mod tidy
  popd;
done
