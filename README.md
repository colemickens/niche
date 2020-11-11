# niche

A tool for managing a cloud "blob" storage container as a binary cache
mirror for serving substitutes to Nix clients.

[![Gitpod ready-to-code](https://img.shields.io/badge/Gitpod-ready--to--code-blue?logo=gitpod)](https://gitpod.io/#https://github.com/colemickens/niche)

## overview

- support (and tested) for pushing to S3, Azure, Wasabi, etc

## documentation

1. Create storage container.
2. Get access credentails.
3. Create config file. (create signing keys)
4. Upload config file.

## why

ghc, boo. proprietary, boo. self-hosting, bandwidth, faster solution for OTF build outputs, etc

## why not

Cachix is great. Cachix is much more flexible and powerful and likely reliable.

## usage examples

```bash
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

## todo

- lots of automated tests
- it might help if we could get post-build-hook for all things, evne if not actually built locally

## thanks

[donate]()
