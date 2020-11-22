#!/usr/bin/env bash
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
set -euo pipefail
set -x

(cd ..)

function niche() { command ../niche "${@}"; }

set -o allexport
source ${DIR}/../.ci/unencrypted/envvars
set +o allexport

export GNUPGHOME="$(mktemp -d nichetest.XXXXXXXX)"
trap "rm -rf $GNUPGHOME" EXIT
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
fp="$(gpg --with-colons --fingerprint --list-keys "$email" | awk -F: '$1 == "fpr" {print $10;}')"

#thing="github:colemickens/flake-impure"
thing="${HOME}/code/flake-impure"
cachename="cache${RANDOM}"
cache="azdev2020nov.blob.core.windows.net/${cachename}"

echo "${AZURE_ACCOUNT}"
echo "${AZURE_KEY}"
niche config init -k azure -c "$cachename" -p "$fp" azdev2020nov
#TODO: trap "niche destroy --yes-really-delete-it $cache" EXIT

# just spam download/upload on the config for sanity checks sake
# TODO: test a version without this step to ensure initial config is perfect
tmpfile="$(mktemp -d nichetestcfg.XXXXXXXX)";
trap "rm -rf $tmpfile" EXIT
niche config download -f "${tmpfile}" "${cache}"
niche config upload -f "${tmpfile}" "${cache}"

# build it so we can grab the outlink (so we can test realization later)
tmpoutlink="$(mktemp -d nichetestout.XXXXXXXX)";
trap "rm -rf $tmpfile" EXIT
nix build --impure --out-link "${tmpoutlink}" "${thing}"
out="$(readlink -f "${tmpoutlink}")"

# now build with `niche` so it's cached
niche build "$cache" -- "${out}"

################################################# TODO: niche build leaks a gc root

#TODO: alternatively test `niche upload`

# remove the GC root
rm /tmp/outlink

# delete the build product from the store
nix-store --delete "${out}"

# now re-acquire the store path by checking our specific cache
nix build "${out}" -j0 \
  --option 'extra-binary-caches' "https://${cache}" \
  --option 'trusted-public-keys' "$(niche show ${cache})"
