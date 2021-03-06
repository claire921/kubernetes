#!/bin/bash

# Copyright 2014 Google Inc. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This script will build and release Kubernetes.

set -eu
set -o pipefail
IFS=$'\n\t'
SCRIPT_DIR=$(CDPATH="" cd $(dirname $0); pwd)

function json_val () {
    python -c 'import json,sys;obj=json.load(sys.stdin);print obj'$1'';
}

source $SCRIPT_DIR/config.sh
source $SCRIPT_DIR/../../cluster/${KUBE_CONFIG_FILE-"config-default.sh"}

$SCRIPT_DIR/../build-release.sh $INSTANCE_PREFIX

if [ -z "$(azure storage account show $STG_ACCOUNT 2>/dev/null | \
    grep data)" ]; then
    azure storage account create -l "$AZ_LOCATION" $STG_ACCOUNT
fi

stg_key=$(azure storage account keys list $STG_ACCOUNT --json | \
    json_val '["primaryKey"]')

if [ -z "$(azure storage container show -a $STG_ACCOUNT -k "$stg_key" \
    $CONTAINER 2>/dev/null | grep data)" ]; then
    azure storage container create \
        -a $STG_ACCOUNT \
        -k "$stg_key" \
        -p Blob \
        $CONTAINER
fi

if [ -n "$(azure storage blob show -a $STG_ACCOUNT -k "$stg_key" \
    $CONTAINER master-release.tgz 2>/dev/null | grep data)" ]; then
    azure storage blob delete \
        -a $STG_ACCOUNT \
        -k "$stg_key" \
        $CONTAINER \
        master-release.tgz
fi

azure storage blob upload \
    -a $STG_ACCOUNT \
    -k "$stg_key" \
    $SCRIPT_DIR/../../output/release/master-release.tgz \
    $CONTAINER \
    master-release.tgz
