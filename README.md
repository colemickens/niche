# niche

[![builds.sr.ht status](https://builds.sr.ht/~colemickens/niche.svg)](https://builds.sr.ht/~colemickens/niche?)

`niche` creates and manages Nix caches backed by cloud blob storage. `niche` wraps `nix build`
to upload build artifacts *as they're produced* rather than waiting for a successful build.

**Tested with**: Azure, B2, S3, Wasabi, ~~Google Storage~~ ([see here]([issue-link](https://github.com/colemickens/niche/issues/4#issuecomment-738495142)))

# Please don't share this, it's got a bug in it still

- [features](#features)
- [install](#install)
    - [~~from nixpkgs~~](#sfrom-nixpkgss)
    - [from source, via flake](#from-source-via-flake)
- [usage](#usage)
  - [create new niche cache](#create-new-niche-cache)
  - [reconfigure niche cache](#reconfigure-niche-cache)
  - [view public key](#view-public-key)
  - [build and upload](#build-and-upload)
- [accessing your cache](#accessing-your-cache)
- [development](#development)
- [thanks](#thanks)

## features

* **wrapped `nix build`** _(upload as you build)_

  `niche build` wraps `nix build` and uploads each build result *as they're produced*. This allows you to cache successful subsets of builds, even if the full build ends up failing.

* **(local-) configuration-free!**

  The configuration for a `niche` cache (including storage credentials and signing key) is stored, encrypted, in the cache itself. Many users will never need to look at it!

* **`sops` - flexible, secure, shared-access to the signing key**

  `niche` uses the [`mozilla/sops`](https://github.com/mozilla/sops) tool, to encrypt and decrypt the configuration on-the-fly. It encrypts the config with multiple keys - (ex: GPG, AGE). It can also encrypt using cloud KMS providers, thereby delegating access control of the signing key to an auditable cloud resource.

* **easy (non-interactive, stateless) usage**

  The handling of key material and configuration is entirely hidden from the user in most cases. Any user with a valid key can upload without any other configuration.

  As an example, consider `builds.sr.ht` where GPG integration is built in: Listing the fingerprint of a key enrolled with `builds.sr.ht` in the config file is all it takes to allow your jobs to be able to upload.

* **bring your own storage (Azure, Amazon, Google, Minio, Wasabi)**

  If `stow` (or rather, [`our fork of stow`](https://github.com/graymeta/stow)) supports it, so do we!

  `niche` can manage Nix mirrors **behind firewalls**, in **Internet-less** virtual networks, leverage **free internal bandwidth** in cloud provider networks, etc.


## install

#### ~~from nixpkgs~~
~~It's available from `nixpkgs` as `niche`.~~

**I would prefer this not be submitted to nixpkgs until the CLI is considered stable.**

#### from source, via flake
1. Install `nix`.
2. Activate [`flakes`]().
3. Add it to your `devenv.nix`, your CI's `shell.nix`, or install it in your profile:
```
nix profile install 'github:colemickens/niche'
```

You can also use this repo as a `nixpkgs` tarball, with `niche` on top. I personally use this so that I can have `niche` and `nixUnstable`
available in a single `nixpkgs` that I can then use on arbitrary build machines with stable `nix-shell`:
```
nix-shell -I nixpkgs=https://github.com/colemickens/niche/archive/master.tar.gz -p niche -p nixUnstable
```

## usage

### create new niche cache
* **`niche config init -k <kind> -f <fingerprint1>[,<fp2>]`** will create an initial signing key and configuration file:
  ```bash
  ❯ export FP="8A94ED58A476A13AE0D6E85E9758078DE5308308"
  ❯ export AZURE_ACCOUNT='azstrg01'
  ❯ export AZURE_KEY='base64lookignstringhere=='
  ❯ export CACHE_NAME="cache"
  ❯ niche config init \
      --name "az.cache.r10e.tech"
      --kind 'azure' \
      --bucket "${CACHE_NAME}" \
      --fingerprints "${FP}"
  ```
  important notes:
  * `--bucket` (`CACHE_NAME`) must be globally unique for all non-Azure providers
  * `--name` (`FRIENDLY_NAME`) is used to form the cache-{priv/public}-key strings (ex: `colescache:yourmirrorpubkeyb64==`)

### reconfigure niche cache
* **`niche config download -u <niche_url> -f <tmp_path>`** downloads and decrypts the configuration file. This allows for manual configuration of the `sops` keygroup or rotating the signing key.
  ```bash
  ❯ niche config download -f '/tmp/tmpnicheconfig' "azstrg01.blob.core.windows.net/cache"
  ```
* **`niche config upload -f <tmp_path>`** re-encrypts the configuration file according to the embedded keygroups. Then uploads the configuration file to it's own bucket.
  ```bash
  ❯ niche config upload -f '/tmp/tmpnicheconfig'
  ❯ shred '/tmp/tmpnicheconfig'
  ```

### view public key
* **`niche show -u <niche_url>`** shows the public key for an existing niche cache:
  ```bash
  ❯ niche show 'azstrg01.blob.core.windows.net/cache'
  az.cache.r10e.tech:thislookslikeaned25519key==

  ❯ niche show 'http://nix.customdomain.com/cache' # same cache, with CNAME
  az.cache.r10e.tech:thislookslikeaned25519key==
  ```
  * (the `https://` is inferred if missing, if you use a domain+HTTP, you must be explicit about it each time)

### build and upload
* **`niche build -u <niche_url> -- [nix build flags]`** wraps `nix build` and uploads new store paths *as they're built*:
  ```bash
  ❯ niche build -u 'azstrg01.blob.core.windows.net/cache' -- \
    '.#hosts.azlinux.config.system.build.toplevel' -j0 --keep-going
  ```

Set `NICHE_DEBUG` to a non-empty value for the most verbose logging out.

## accessing your cache

This step depends on how you choose to have your users acces your storage provider (`kind`), and depends on if you use a custom domain.

Most of the cloud storage providers allow for a custom domain pointed at a bucket. However, this usually requires falling back to HTTP. Some providers also offer a CDN service that can allow HTTPS with custom domains.

The following are examples of value would be used as `<niche_url>` in the usage:
* azure:
  * `https://$AZURE_ACCOUNT_NAME.blob.core.windows.net/${CACHE_NAME}`
  * `azstrg01.blob.core.windows.net/cache` (per the example above, omitting the optional `https://` prefix)
  * `http://cache.r10e.dev/cache` (where `cache.r10e.dev` is a CNAME to the same full storage url, note the **http**)
  * `nixcache.r10e.dev` (using Azure CDN for SSL)
* b2:
  * `https://s3.${B2_REGION}.backblazeb2.com/${CACHE_NAME}`
* ~~google:~~ ([see here]([issue-link](https://github.com/colemickens/niche/issues/4#issuecomment-738495142)))
  * ~~`storage.cloud.google.com/${CACHE_NAME}`~~
* s3:
  * `https://${CACHE_NAME}.s3.amazonaws.com`
  * `http://s3cache.r10e.dev` (using CNAME)
  * `s3cache.r10e.dev` (using CloudFront)
* wasabi:
  * `s3.wasabisys.com/${CACHE_NAME}`

## development

Development of `niche` is done with `nix`:

```shell
❯ git clone https://github.com/colemickens/niche
❯ nix build '.#'
❯ ./result/bin/niche --version # TODO: add a version command and log it during startups for bug report purposes
dirty
# TODO fix this up
```

## thanks

~~[donate]()~~

([left over readme](./README_EX.md))
