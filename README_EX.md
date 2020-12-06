## big idea

### Investigate another mode:
  - build the derivation
  - walk all the inputs, etc, upload ones that are built
    - this could let us catch things that we miss between the p-b-h (see issue on nixos/nix)


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

10. Add an example where you publish to a $web dir in azure so it can be
    on the root of a subdomain, instead of after some basepath.

12. Should we have more threads processing uploads?
13. Should we 'skipping already processed path' ahead of sending it over the socket?



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