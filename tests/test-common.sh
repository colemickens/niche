#!/usr/bin/env bash
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
set -o allexport; source "${DIR}/../.ci/unencrypted/envvars_${kind}"; set +o allexport
function niche() { command ../niche "${@}"; }

set -euo pipefail
set -x

export GNUPGHOME="$(mktemp)"; trap "rm -rf $GNUPGHOME" EXIT
rm -rf "$GNUPGHOME"
mkdir -p -m 700 "$GNUPGHOME"
email="testperson@example.com"
cat<<-EOF | gpg --batch --gen-key
%no-protection
Key-Type: RSA
Key-Length: 1024
Name-Real: testperson
Name-Email: testperson@example.com
EOF
gpg --export "$email" > $GNUPGHOME/test.gpg
fp="$(gpg --with-colons --fingerprint --list-keys "$email" \
  | awk -F: '$1 == "fpr" {print $10;}')"

niche config init -k "${kind}" -c "$cachename" -p "$fp" azdev2020nov
#TODO: trap "niche destroy --yes-really-delete-it $cache" EXIT # TODO?

# build it so we can grab the outlink (so we can test realization later)
outlink="$(mktemp -d)"; rm -rf "${outlink}"; trap "rm -rf $outlink" EXIT
nix build --impure --out-link "${outlink}" "${thing}"

# we can't read it to a variable, so stash the out path in a file
# -> if we do, it is the environ for the `niche` process and it seems like somehow
#    that infects bash and then prevents nix-store --delete from working
#    (if you don't believe this, try to change it...)
ttt=$(mktemp); trap "rm -rf $ttt" EXIT
readlink -f "${outlink}" > $ttt
rm -rf "${outlink}"

# now build with `niche` so it gets signed+uploaded to our cache
niche build "$cache" -- "$(cat $ttt)"

#TODO: alternatively test `niche upload`

# delete the build product from the store
nix-store --delete "$(cat $ttt)" \
  || nix-store --gc --print-roots | rg bundle || true

publickey="$(niche show ${cache})"

# make sure it really went away
(ls -al "$(cat $ttt)" && false) || true

# now re-acquire the store path by checking our specific cache
outlink="$(mktemp -d)"; rm -rf "${outlink}"; trap "rm -rf $outlink" EXIT
nix build "$(cat $ttt)" -j0 \
  --option 'extra-binary-caches' "https://${cache}" \
  --option 'trusted-public-keys' "${publickey}" \
  --out-link "${outlink}"

# prove it's really back (aka, was cached and had a good signature)
ls -al $(readlink "${outlink}")
