#!/usr/bin/env bash
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
set -euo pipefail
set -x

(cd "${DIR}/.."; go build .); cd "${DIR}"

export thing="github:colemickens/flake-impure"
export cachename="cache${RANDOM}"
export cache="s3.wasabisys.com/${cachename}"
export kind="s3"

export variant="wasabi"

./test-common.sh
