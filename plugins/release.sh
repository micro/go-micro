#!/bin/bash

tag=$1
commitsh=$2

if [ "x$tag" = "x" ]; then
  echo "must specify tag to release"
  exit 1;
fi

for m in $(find * -name 'go.mod' -exec dirname {} \;); do
  if [ ! -n "$commitsh" ]; then
    hub release create -m "plugins/$m/$tag release" plugins/$m/$tag;
  else
    hub release create -m "plugins/$m/$tag release" -t $commitsh plugins/$m/$tag;
  fi
done