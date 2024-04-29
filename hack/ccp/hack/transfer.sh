#!/usr/bin/env bash

set -x

__dir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

transfer=$(mktemp -d)
cp $__dir/processingMCP.xml $transfer
touch $transfer/hello.txt
touch $transfer/bye.txt
mv $transfer ~/.ccp/am-pipeline-data/watchedDirectories/activeTransfers/standardTransfer/
