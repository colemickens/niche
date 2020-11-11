# niche

A tool for managing a cloud "blob" storage container as a binary cache
mirror for serving substitutes to Nix clients.

## overview

- support (and tested) for pushing to S3, Azure, Wasabi, etc

## documentation

1. Create storage container.
2. Get access credentails.
3. Create config file. (create signing keys)
4. Upload config file.

## usage examples

```bash
# interact with the config stored (encrypted) in the cached
niche config init # wizard for wiring up existing storage, or creating a new one + key
niche config update -c ./config.json # force upload/overwrite the config for a repo

# this will start the listener and then call nix build with the post-build-hook
# to send to the listener we start. we can hide all of that from the user
niche cache -h 'https://cache.niche.org' '.#hosts.azlinux.config.system.build.toplevel'
niche cache -h 'https://cache.niche.org' '.#hosts.azlinux.config.system.build.toplevel' -- -j0 --extra-binary-substittute....

# or, maybe they want to have one upload thread and... run multiple nix builds, ok?
niche listen -s '/tmp/somesocket.sock' -h 'https://cache.niche.org'

niche queue -s '/tmp/somesocket.sock' '.#some.drv'
niche queue -s '/tmp/somesocket.sock' '.#some.drv' # todo: how does nix cli detect if it's a "thing" or a store path?
```

## thanks

[donate]()
