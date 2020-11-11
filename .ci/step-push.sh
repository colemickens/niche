#!/usr/bin/env bash
set -euo pipefail
set -x

ssh-keyscan github.com >> ${HOME}/.ssh/known_hosts

# TODO: load GitHub PAT from unencrypted secret

git add -A .
git diff-index --cached --quiet HEAD || git commit -m "auto-updates"
git push origin HEAD
