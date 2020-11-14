# niche

[![builds.sr.ht status](https://builds.sr.ht/~colemickens/niche-azure.svg)](https://builds.sr.ht/~colemickens/niche-azure?)

`niche` uploads store paths to your binary cache mirror running in any cloud blob storage. It can also wrap `nix build`
to upload build artifacts *as they're produced* rather than waiting for a successful build.

## demo
(or maybe a GIF with a note that it links to a scrollable asciinema)
[link to screenshot of asciinema]


## features

### wrapped `nix build`
`niche build` wraps `nix build` and uploads each build result *as they're produced*. This allows you to cache intermediate builds,
even if the full build result doesn't end up finishing successfully.

### signing key + cloud credentials protected by `sops`
The configuration for a `niche` cache (including cloud credentials and the signing key)
is stored, encrypted, in the cache itself. `niche` uses [`sops`](https://github.com/mozilla/sops)
to encrypt/decrypt on the fly, as necessary.

This means `niche` comes with automatic delegation and auditing by using `sop`'s KMS/KeyVault integration, or
automatic non-cloud team key sharing by listing your team's GPG keys.

### bring your own storage
AWS S3, Azure Blob Storage, Google Cloud Storage, Wasabi, Minio? **Yes!**
If [`stow`](https://github.com/graymeta/stow) supports it, so do we (in theory).

## usage

**`niche init`** will create a signing key and create an initial configuration file.

```bash
# initialize a configuration file (creates a signing key for you)
niche init -k 'azure' -f '4774EB1BF21D57E93221CF0262556A61E301DC21' -c ./config.json

# initialize the cache by uploading our (encrypted) config
niche reconfigure -u 'https://azdev2020nov.blob.core.windows.net/cache' -c ./config.json

# invoke `nix build` and upload all build results as they're built
niche build 'https://azdev2020nov.blob.core.windows.net/cache' -- \
  '.#hosts.azlinux.config.system.build.toplevel' -j0 --keep-going

# or, just upload a bunch of paths, right now (TODO: DOESNT WORK YET)
storePath="/nix/store/p9vy4sgsh6m2kgph9i2mv3qgr3iy1afc-firefox-82.0.2"
echo $storePath | niche upload "azdev2020nov.blob.core.windows.net/cache"
```

You may need to create a config file by hand:
- if you want to configure the Sops encryption key groups manually, or,
- if you have existing signing keys you want to use instead of generating new ones

## suggested contributions

If you're interested in contributing to `niche`, here are some suggestions:

0. Improve error handling. Not sure what state of the art is in Golang today.
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
5. Come up with a better project "tagline" to use as the first line of text + repo description
6. Add a mode where a user can `listen` and then `queue` from separate processes (one `listen` process, many queues triggering multiple simultaenous `nix build`s, for example)
7. Figure out the right way to plumb `nix build` output to the screen, since the user wants to monitor the build still

## long-term musings

**sops**: While I like the *model* of Sops, I don't like it's overall feature-set, UX, or code quality.
In a dream world, Sops would get a RIIR, with a more limited feature-set, akin, to `git-crypt`, and with good `age` support. Then `niche` would follow.

**stow**: I wish that `stow` had a proper `Provider` struct that could support `Kind()`, and `ConfigFields()->[]string`.
And then the toplevel package should have `Providers()` or `Provider()`. Then there could be an interface `ConfigWithMap()` to take a `map[string]string`, etc.
This would nicely increase the ability to make tooling around Stow itself.

## todo (pre-release)

- figure out why xz stream is failing at times
- better logging, remove fmt.Println()
- lots of automated tests
- it might help if we could get post-build-hook for all things, evne if not actually built locally
- simpler "echo /nix/store/x | niche push foo.bar.com" invocation like cachix

## thanks

~~[donate]()~~
