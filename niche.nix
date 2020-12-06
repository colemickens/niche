{ stdenv, buildGoModule, fetchFromGitHub
, wrapGAppsHook
}:

let metadata = import ./metadata.nix; in
buildGoModule rec {
  pname = "niche";
  version = self.shortRev;

  # patch nicheVersion into the binary, so that we know the version
  # make sure we log it on startup

  src = ./.;
  vendorSha256 = stdenv.lib.fakeSha256;

  subPackages = [ "." ];

  meta = with stdenv.lib; {
    homepage = "https://github.com/colemickens/niche";
    description = "a self-service nix binary cache tool that manages your signing key and wraps nix build to upload build products";
    license = licenses.mit;
    maintainers = with maintainers; [ colemickens ];
    platforms = platforms.linux;
  };
}
