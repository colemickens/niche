#!/usr/bin/env bash
set -euo pipefail
set -x

# update all deps
go get -u ./...

# see if we still build
go build ./...

# nixpkgs from flake.lock
nixpkgs="https://api.github.com/repos/$(jq -r '.nodes.nixpkgs.locked.owner' flake.lock)/$(jq -r '.nodes.nixpkgs.locked.repo' flake.lock)/tarball/$(jq -r '.nodes.nixpkgs.locked.rev' flake.lock)"

# update our own modSha256 in `flake.nix`
vendorSha256="$(grep vendorSha256 flake.nix | cut -d'"' -f2)"
newvendorSha256="$(NIX_PATH="nixpkgs=${nixpkgs}" \
  nix-prefetch \
    "{ sha256 }: let p=(import ./packages.nix).niche; in p.go-modules.overrideAttrs (_: { vendorSha256 = sha256; })")"
sed -i "s|${vendorSha256}|${newvendorSha256}|" "flake.nix"
