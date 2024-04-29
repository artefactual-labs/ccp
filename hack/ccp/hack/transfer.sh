#!/usr/bin/env bash

set -x

transfer=$(mktemp -d)
touch $transfer/hello.txt
touch $transfer/bye.txt
mv $transfer ~/.ccp/am-pipeline-data/watchedDirectories/activeTransfers/standardTransfer/
