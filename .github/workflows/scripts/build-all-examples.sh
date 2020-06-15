#!/bin/bash
# set -x

function build_binary {
    echo building $1
    pushd $1
    go build -o _main
    local ret=$?
    if [ $ret -gt 0 ]; then 
        failed=1
        failed_arr+=($1)
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
failed_arr=()
failed=0
go mod edit -replace github.com/micro/go-micro/v2=github.com/micro/go-micro/v2@$1 
check_dir . $1
if [ $failed -gt 0 ]; then
    echo Some builds failed
    printf '%s\n' "${failed_arr[@]}"
fi
exit $failed