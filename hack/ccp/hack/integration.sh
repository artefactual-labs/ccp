#!/usr/bin/env bash

# USAGE:
# ./hack/integration.sh "TestServerCreatePackageWithUserDecision"

set -e

readonly __dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly __root="$(cd "$(dirname "${__dir}")" && pwd)"

runFlag=""
if [[ -n "$1" ]]; then
    runFlag="-run=$1"
fi

env \
    CCP_INTEGRATION_ENABLED=1 \
    CCP_INTEGRATION_ENABLE_LOGGING=yes \
    CCP_INTEGRATION_ENABLE_TESTCONTAINERS_LOGGING=no \
    CCP_INTEGRATION_ENABLE_MCPCLIENT_LOGGING=no \
        go test -count=1 -v ${__root}/integration/ $runFlag
