# niche

A tool for managing a cloud "blob" storage container as a binary cache
mirror for serving substitutes to Nix clients.

[![Gitpod ready-to-code](https://img.shields.io/badge/Gitpod-ready--to--code-blue?logo=gitpod)](https://gitpod.io/#https://github.com/colemickens/niche)

## overview

- support (and tested) for pushing to S3, Azure, Wasabi, etc
- we know nothing about the url, right? just storage b/e details
- maybe we need the URL for forming the nar url in the narinfo tho?

## why this is cool

1. It wraps `nix build` and catches built packages on the fly. This is highly useful for CI scenarios.
2. You don't have to worry about managing your Nix signing key. You can delegate protecting it to Sops (and thereby to GPG or cloud authentication where Sops supports it)
3. You can bring-your-own-storage! This is useful for all sorts of security- or cost-conscious reasons.

## compared to cachix
- cachix is free, cachix is a managed service
- cachix is pricey, cachix is not on-prem
- cachix doesn't (yet) handle OTF build outputs very well

## documentation

1. Create storage container.
2. Get access credentails.
3. Create signing keys.
4. Create config file.
5. Upload config file.

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

## long-term

While I like the *model* of Sops, I don't like it's overall feature-set, UX, or code quality.
I think it's ripe for a Rust/(r)age-y replacement.

I'd like to replace Sops and then rewrite this in Rust using "Rops" or whatever it might be.

## todo

- lots of automated tests
- it might help if we could get post-build-hook for all things, evne if not actually built locally

## thanks

[donate]()
