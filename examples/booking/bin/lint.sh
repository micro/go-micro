#!/usr/bin/env bash

dir=`pwd`

check() {
	for d in $(ls ./$1); do
		echo "verifying $1/$d"
		pushd $dir/$1/$d >/dev/null
		go fmt
		golint
		popd >/dev/null
	done
}

check api
check srv
