#!/usr/bin/env bash
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
set -o allexport; source ${DIR}/../.ci/unencrypted/envvars; set +o allexport
set -euo pipefail
set -x

(cd ..; go build .)

export thing="github:colemickens/flake-impure"
export cachename="cache${RANDOM}"
export cache="${cachename}.s3.amazonaws.com"
export kind="s3"
./test-common.sh
