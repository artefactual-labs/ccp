#!/usr/bin/env bash

set -e

readonly __dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly __root="$(cd "$(dirname "${__dir}")" && pwd)"

env \
    CCP_INTEGRATION_ENABLED=1 \
    CCP_INTEGRATION_ENABLE_LOGGING=no \
    CCP_INTEGRATION_ENABLE_TESTCONTAINERS_LOGGING=no \
        go test -count=1 -v ${__root}/integration/ -run=TestServerCreatePackage
