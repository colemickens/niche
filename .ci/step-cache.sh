#!/usr/bin/env bash
set -euo pipefail
set -x

cache="niche"

#set +x;
export CACHIX_SIGNING_KEY="$(cat '.ci/unencrypted/cachix_niche_signing_key' | head -1)"
#set -x

nix --experimental-features 'nix-command flakes' build .
readlink -f result | cachix push "${cache}"
