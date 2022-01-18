#!/bin/bash

DIR=/tmp/services
CDIR=`pwd`
REPO=https://github.com/m3o/m3o-go

git clone $REPO $DIR
cd $DIR

# move/delete
rm -rf .git .github go.mod go.sum README.md CNAME Makefile TODO.md LICENSE client cmd examples
mv m3o.go services.go

# rewrite
grep -r "go.m3o.com" | cut -f 1 -d : | xargs sed -i 's@go.m3o.com/client@go-micro.dev/v4/api/client@g'
sed -i 's@go.m3o.com@go-micro.dev/v4/services@g' services.go
sed -i 's@package m3o@package services@g' services.go

# sync
cd $CDIR
rsync -avz $DIR/ .

# cleanup
rm -rf $DIR
