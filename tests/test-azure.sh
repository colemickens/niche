#!/usr/bin/env bash
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
set -euo pipefail
set -x

(cd ..; go build .)

thing="github:colemickens/flake-impure"
cachename="cache${RANDOM}"
cache="azdev2020nov.blob.core.windows.net/${cachename}"
kind="azure"

./test-common.sh
