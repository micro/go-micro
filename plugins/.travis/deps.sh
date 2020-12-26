#!/bin/bash -ex

PKGS=""
for d in $(find * -name 'go.mod'); do
  pushd $(dirname $d)
  go mod download
  popd
done
