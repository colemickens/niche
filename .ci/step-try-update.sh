#!/usr/bin/env bash
set -euo pipefail
set -x

# update all deps
go get -u ./...

# ensure our stow fork is up-to-date
stowrev="$(git ls-remote https://github.com/colemickens/stow --rev HEAD | cut -d"$(echo -e \\t)" -f1)"
go mod edit -replace "github.com/graymeta/stow=github.com/colemickens/stow@${stowrev}"

# tidy (this also resolves the above to a rev, though that might be cached??)
go mod tidy

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
