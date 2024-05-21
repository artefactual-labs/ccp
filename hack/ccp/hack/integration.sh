#!/usr/bin/env bash

set -e

readonly __dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly __root="$(cd "$(dirname "${__dir}")" && pwd)"

docker compose stop archivematica-ccp 2>/dev/null

CCP_DIR=${HOME}/.ccp
CCP_AM_DIR=${CCP_DIR}/am-pipeline-data
CCP_SS_DIR=${CCP_DIR}/ss-location-data

env \
    CCP_INTEGRATION_ENABLED=1 \
    CCP_INTEGRATION_TRANSFER_SOURCE=${CCP_SS_DIR}/archivematica/ \
    CCP_INTEGRATION_USE_COMPOSE=yes \
    CCP_INTEGRATION_USE_STDOUT=yes \
        go test -count=1 -v ${__root}/integration/ -run=TestServerCreatePackage
