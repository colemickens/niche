#!/usr/bin/env bash
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
set -euo pipefail
set -x

(cd "${DIR}/.."; go build .); cd "${DIR}"

export thing="github:colemickens/flake-impure"
export cachename="cache${RANDOM}"
export cache="s3.us-west-002.backblazeb2.com/${cachename}"
export kind="b2"
./test-common.sh
