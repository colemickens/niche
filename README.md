# niche

[![builds.sr.ht status](https://builds.sr.ht/~colemickens/niche-azure.svg)](https://builds.sr.ht/~colemickens/niche-azure?)

`niche` uploads store paths to your binary cache mirror running in any cloud blob storage. It can also wrap `nix build`
to upload build artifacts *as they're produced* rather than waiting for a successful build.

**Tested with**: Azure, B2, Google Storage, S3, Wasabi

(**Warning**: `niche` does no locking currently. I'm not sure what happens if clients concurrently upload a path.)

*If you find this valuable, **please** let me know. Even if it's just a star or an email.*

## features

* **wrapped `nix build`** _(upload as you build)_

  `niche build` wraps `nix build` and uploads each build result *as they're produced*. This allows you to cache partial builds, even if the full build ends up failing.

* **(local-) configuration-free!**

  The configuration for a `niche` cache (including storage credentials and signing key) is stored, encrypted, in the cache itself. Many users will never need to look at it!

* **`sops` - flexible, secure, shared-access to the signing key**

  `niche` uses the [`mozilla/sops`](https://github.com/mozilla/sops) tool, to encrypt and decrypt the configuration on-the-fly. It encrypts the config with multiple keys like GPG fingerprints. It can also encrypt using cloud KMS providers, thereby delegating access control to an auditable cloud resource.

* **easy (non-interactive, stateless) usage**

  The handling of key material and configuration is entirely hidden from the user in most cases. Any user with a valid key can upload without any other configuration.

  As an example, consider `builds.sr.ht` where GPG integration is built in: Listing the fingerprint of a key enrolled with `builds.sr.ht` in the config file is all it takes to allow your jobs to be able to upload.

* **bring your own storage (Azure, Amazon, Google, Minio, Wasabi)**

  If [`stow`](https://github.com/graymeta/stow) supports it, so do we (in theory)!

  `niche` can manage Nix mirrors **behind firewalls**, in **Internet-less** virtual networks, leverage **free internal bandwidth** in cloud provider networks, etc.

## usage

* **`niche config init -k <kind> -f <fingerprint1>[,<fp2>]`** will create an initial signing key and configuration file:
  ```bash
  ❯ export FP="8A94ED58A476A13AE0D6E85E9758078DE5308308" # change
  ❯ export AZURE_ACCOUNT='azstrg01'                      # change
  ❯ export AZURE_KEY='base64lookignstringhere=='         # change
  ❯ niche config init -k 'azure' -f "${FP}"
  ```

* **`niche show <repo>`** shows the public key for an existing niche repo:
  ```bash
  ❯ niche show 'azstrg01.blob.core.windows.net/cache'
  az.cache.r10e.tech:thislookslikeaned25519key==
  ```

* **`niche build <repo> -- [nix build flags]`** wraps `nix build` and uploads new store paths *as they're built*:
  ```bash
  ❯ niche build 'https://azstrg01.blob.core.windows.net/cache' -- \
    '.#hosts.azlinux.config.system.build.toplevel' -j0 --keep-going
  ```

* **`echo [<path>\n...] | niche upload <repo>`** uploads paths piped over stdin:
  ```bash
  ❯ nix build 'github:nixos/nixpkgs/nixos-unstable#firefox' --out-link '/tmp/outlink1'
  ❯ nix build 'github:nixos/nixpkgs/nixos-unstable#neovim' --out-link '/tmp/outlink2'
  ❯ echo $'/tmp/outlink1\n/tmp/outlink2' | niche upload "azstrg01.blob.core.windows.net/cache"
  ```

* **`niche upload [<path>...]`** uploads paths passed as args
  ```bash
  ❯ nix build github:nixos/nixpkgs/nixos-unstable#emacs --out-link /tmp/outlink1
  ❯ nix build github:nixos/nixpkgs/nixos-unstable#neovim --out-link /tmp/outlink2
  ❯ niche upload "azstrg01.blob.core.windows.net/cache" /tmp/outlink1 /tmp/outlink2
  ```

* **`niche config`** commands help download and overwrite a repo's config file.
  This allows for custom `sops` keygroup configuration, override of keys, etc.
  ```bash
  ❯ niche config download -f '/tmp/tmpnicheconfig' "azstrg01.blob.core.windows.net/cache"
  ❯ nvim '/tmp/tmpnicheconfig'
  ❯ niche config upload -f '/tmp/tmpnicheconfig'
  ````

Set `NICHE_DEBUG` to a non-empty value for the most verbose logging out.

## suggested contributions

If you're interested in contributing to `niche`, here are some suggestions:

0. Separate the queue for checking narinfo and doing upload?

0. Suggest Age usage instead of GPG? Ask ghc about this

0. SOme of the log handling is dumb, I recreated the same builder start over and over.
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
8. Might be worth pulling `stow` source into the tree, keeping it updated better, etc

Fix this: https://github.com/NixOS/nix/issues/4294


## long-term musings

**sops**: While I like the *model* of Sops, I don't like it's overall feature-set, UX, or code quality.
In a dream world, Sops would get a RIIR, with a more limited feature-set, akin, to `git-crypt`, and with good `age` support. Then `niche` would follow.

**stow**: I wish that `stow` had a proper `Provider` struct that could support `Kind()`, and `ConfigFields()->[]string`.
And then the toplevel package should have `Providers()` or `Provider()`. Then there could be an interface `ConfigWithMap()` to take a `map[string]string`, etc.
This would nicely increase the ability to make tooling around Stow itself.

## todo (pre-release)

- lots of automated tests
- it might help if we could get post-build-hook for all things, evne if not actually built locally
- simpler "echo /nix/store/x | niche push foo.bar.com" invocation like cachix

## thanks

~~[donate]()~~
