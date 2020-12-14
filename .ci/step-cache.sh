#!/usr/bin/env bash
set -euo pipefail
set -x

cache="niche"

cd ..
nix build .
readlink -f result | cachix push "${cache}"
