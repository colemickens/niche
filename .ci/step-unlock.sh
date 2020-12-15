#!/usr/bin/env bash
set -euo pipefail
set -x

cd .ci

mkdir -p unencrypted;
mkdir -p encrypted; cd encrypted
for f in *; do
  sops \
    --input-type binary --output-type binary \
    --verbose --output ../unencrypted/$f -d $f
done
