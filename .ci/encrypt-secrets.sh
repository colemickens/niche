#!/usr/bin/env bash
set -euo pipefail
set -x

mkdir -p encrypted;
mkdir -p unencrypted; cd unencrypted
for f in *; do
  sops \
    --input-type binary --output-type binary \
    --verbose --output ../encrypted/$f -e $f
done
