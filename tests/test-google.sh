#!/usr/bin/env bash
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
set -euo pipefail
set -x

(cd ..; go build .)

export thing="github:colemickens/flake-impure"
export cachename="cache${RANDOM}"
export cache="storage.cloud.google.com/${cachename}"
export kind="google"

./test-common.sh
