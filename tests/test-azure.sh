#!/usr/bin/env bash
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
set -euo pipefail
set -x

(cd "${DIR}/.."; go build .); cd "${DIR}"

export thing="github:colemickens/flake-impure"
export cachename="cache${RANDOM}"
export cache="azdev2020nov.blob.core.windows.net/${cachename}"
export kind="azure"
./test-common.sh
