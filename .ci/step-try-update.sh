#!/usr/bin/env bash
set -euo pipefail
set -x

# update all deps
go get -u ./...

# see if we still build
go build ./...

# update our own modSha256 in `flake.nix`

