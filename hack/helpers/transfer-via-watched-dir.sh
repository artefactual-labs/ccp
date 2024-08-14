#!/usr/bin/env bash

set -x

__dir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

transfer=$(mktemp -d)

# cp $__dir/processingMCP.xml $transfer

num_files=$1
if [[ ! "$num_files" =~ ^[0-9]+$ ]]; then
    touch "$transfer/hello.txt"
    touch "$transfer/bye.txt"
else
    for (( i=0; i<num_files; i++ )); do
        mktemp --tmpdir="$transfer" XXXXXXXX.txt
    done
fi


mv $transfer ~/.ccp/am-pipeline-data/watchedDirectories/activeTransfers/standardTransfer/
