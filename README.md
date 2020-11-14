# niche

A tool for managing a cloud "blob" storage container as a binary cache
mirror for serving substitutes to Nix clients.

**ALPHA QUALITY**: config may change, this may silently upload corrupted archive, etc, yatta yatta
[![builds.sr.ht status](https://builds.sr.ht/~colemickens/niche-azure.svg)](https://builds.sr.ht/~colemickens/niche-azure?)

## demo

(or maybe a GIF with a note that it links to a scrollable asciinema)
[link to screenshot of asciinema]


## features

### wrapped `nix build`
Most usages of `nix` mean that build products are not known if the build fails. If you are in a CI scenario, build 90%
of the universe and then fail, you don't want to rebuild that 90% again. The only way to do this today is by calling
`nix` with a post-build-hook and then orchestrating the compress+sign+upload process. **Instead `niche` does all of this for you.**

### signing key + cloud credentials protected by `sops`
`niche` uses [`sops`](https://github.com/mozilla/sops) for encrypting its own configuration (including backend storage creds + signing key).
This means you can easily allow others access to your signing key by adding their GPG key to the configuration and re-uploading.

This also has the unique featuring of being able to delegate access control to the signing key to the Access Management platform
of your particular cloud. For example, it's very easy to tell `niche`/`sops` to use an Azure KeyVault to encrypt/decrypt
the configuration file. Then you can control what users are able to access that KeyVault instance (and thereby the signing key and storage account key).

Alternatively, if you're a single user or not on a cloud platform, it's very straight-forward to use GPG. This is what the
`niche` repo itself does for the configurations for its automated testing. The configuration files are configured to be
encrypted by **@colemickens**'s GPG key, as well as the GPG key configured for their `builds.sr.ht` builds to utilize.
(This also meant that I was able to just immediately start using `niche` from my CI by adding a call to `niche`, no need
to muck with plumbing in additional credentials!)

### bring your own storage
AWS S3, Azure Blob Storage, Google Cloud Storage, Wasabi, Minio? Whatever!

**NOTE**: Only *Azure* has (funding for) automated testing. If you can fund/sponsor or otherwise provide access to
other platforms, I can add automated tests and an example configuration.

If [`stow`](https://github.com/graymeta/stow) supports it, so do we (in theory).

## usage

## create intial configuration file

This file tells `niche` how to access your cloud storage to check/upload files,
and also stores the signing key that `niche` uses to sign NARs before uploading.

This file is encrypted using the `keyGroups` in the config, using `sops`.
`niche` internally handles encrypting and decrypting this file as needed. When
it is stored on the server, it is always encrypted.

```bash
mkdir /tmp/keys
cd /tmp/keys

HOST="az.cache.r10e.tech"

nix-store --generate-binary-cache-key "${HOST}" priv pub

PRIV="$(cat /tmp/keys/priv | head -1)"
PUB="$(cat /tmp/keys/pub | head -1)"

cat<<EOF >/tmp/azure.json
{
  "signingKey": "$PRIV",
  "publicKey": "$PUB",
  "storageKind": "azure",
  "storageContainer": "cache",
  "storageConfigMap": {
    "account": "azdev2020nov",
    "key": "$AZ_STORAGE_KEY"
  },
  "keyGroups": [{
    "pgp": [ "$MY_GPG_FINGERPRINT" ]
  }]
}
EOF
```

# initialize or reconfigure the specified cache
# (if no configFile is passed, the existing config is opened for viewing/editing)
niche reconfigure -u 'https://nix.example.org/cache' -c ./config.json

# this wraps `nix build`, and uses a post-build-hook to cache all outputs as they're produced
niche build -u 'https://cache.niche.org' -- '.#hosts.azlinux.config.system.build.toplevel' -j0 --keep-going

# or, maybe they want to have one upload thread and... run multiple nix builds, ok?
niche listen -u 'https://cache.niche.org' -s '/tmp/somesocket.sock'
echo /nix/store/abcdefghi-foo | niche queue -s '/tmp/somesocket.sock'

# or, auto-sock
SOCK=$(niche listen -u 'https://cache.niche.org')
echo /nix/store/abcdefghi-foo | niche queue -s "${SOCK}"

# or, just upload a bunch of paths, right now
echo /nix/store/abcdefghi-foo | niche upload -u 'https://nix.example.org/cache'

# TODO: multi-line-output example/test
```

## suggested contributions

If you're interested in contributing to `niche`, here are some suggestions:
0. Add a `niche init` option that walks the user through initial config file creation
1. Reduce the places where we need to call out to `nix` (all usages are in `nix.go`)
    * `nix dump-path <path>` (get the NAR stream to compress + upload)
    * `nix path-info <path>` (initially populate the fields of a narInfo struct)
    * `nix to-base32 <path>` (convert hashes at edges back into base32 for fingerprinting)
    * `nix-store -q -R <path>` (get all child paths, used when no builds occur and we just see the final store path)
    * I doubt we'd want to or achieve removing `nix build ...` for the inner build process
2. I think most of the `TODO`s should be approachable
3. Make it so that we only expand store paths when it is the last thing sent (special token?)
4. Filter the list of store paths to process and ignore them if we already processed them in this session.

## long-term

While I like the *model* of Sops, I don't like it's overall feature-set, UX, or code quality.
I think it's ripe for a Rust/(r)age-y replacement.

I'd like to replace Sops and then rewrite this in Rust using "Rops" or whatever it might be. The goal would
be to keep the CLI 100% compatible. Which should be easy, it's very simple.

## todo (pre-release)

- better logging, remove fmt.Println()
- lots of automated tests
- it might help if we could get post-build-hook for all things, evne if not actually built locally
- simpler "echo /nix/store/x | niche push foo.bar.com" invocation like cachix

## thanks

[donate]()
