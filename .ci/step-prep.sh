#!/usr/bin/env bash
set -euo pipefail
set -x

cachix use colemickens

# not sure this is needed with https git remote
mkdir -p "${HOME}/.ssh"
ssh-keyscan github.com >> ${HOME}/.ssh/known_hosts
# would be nice to have this the same across srht jobs

git config --global user.name \
 "Cole Botkens"

git config --global user.email \
 "cole.mickens+colebot@gmail.com"

# first things first, let's update our flake
nix --experimental-features 'nix-command flakes' flake update --recreate-lock-file --no-registries
(git add -A . && git commit -m "auto-update: flake.lock") || true
