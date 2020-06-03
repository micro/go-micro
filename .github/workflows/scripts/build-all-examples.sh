#!/bin/bash
# set -x

function build_binary {
    echo building $1
    pushd $1
    go mod init example
    go mod edit -require=github.com/micro/go-micro/v2@$2
    go build
    local ret=$?
    if [ $ret -gt 0 ]; then 
        failed=1
    fi
    popd
}

function is_main {
    grep "package main" -l -dskip $1/*.go > /dev/null 2>&1
}


function check_dir {
    is_main $1
    local ret=$?
    if [ $ret == 0 ]; then
        build_binary $1 $2
    fi
    for filename in $1/*; do
        if [ -d $filename ]; then
            check_dir $filename $2
        fi
    done
}

failed=0
this_hash=$1
check_dir . $1
exit $failed