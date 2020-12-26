#!/bin/bash

tag=$1

if [ "x$tag" = "x" ]; then
  echo "must specify tag to release"
  exit 1;
fi

for m in $(find * -name 'go.mod' -exec dirname {} \;); do
  hub release create -m "$m/$tag release" $m/$tag;
done
