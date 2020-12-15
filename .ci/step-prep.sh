#!/usr/bin/env bash
set -euo pipefail
set -x

cachix use colemickens

git config --global user.name \
 "Cole Botkens"

git config --global user.email \
 "cole.mickens+colebot@gmail.com"

echo "https://colebot:$(cat .ci/unencrypted/github_niche_ci_pat | head -1)@github.com" \
  > "${HOME}/.git-credentials"

git config credential.helper 'store'

# first things first, let's update our flake
nix --experimental-features 'nix-command flakes' flake update --recreate-lock-file --no-registries
(git add -A . && git commit -m "auto-update: flake.lock") || true
