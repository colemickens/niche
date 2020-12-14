{
  description = "niche";

  inputs = {
    nixpkgs = { url = "github:nixos/nixpkgs/nixos-unstable"; };
  };

  outputs = inputs:
    let
      nameValuePair = name: value: { inherit name value; };
      genAttrs = names: f: builtins.listToAttrs (map (n: nameValuePair n (f n)) names);
      supportedSystems = [ "x86_64-linux" "aarch64-linux" ];
      forAllSystems = genAttrs supportedSystems;

      pkgsFor = pkgs: sys: import pkgs {
        system = sys;
        config = { allowUnfree = true; };
      };
      pkgs_ = genAttrs (builtins.attrNames inputs) (inp: genAttrs supportedSystems (sys: pkgsFor inputs."${inp}" sys));

      nichePkg = { stdenv, buildGoModule, fetchFromGitHub }:
        let metadata = import ./metadata.nix; in
        buildGoModule rec {
          pname = "niche";
          version = inputs.self.shortRev or "dirty";
          src = ./.;
          #vendorSha256 = "sha256-OZ73SI+2Vqk+NLyhyQami88kiMhhwmTb9A8LArlBMJE=";
          vendorSha256 = stdenv.lib.fakeSha256;
          subPackages = [ "." ];
          meta = with stdenv.lib; {
            homepage = "https://github.com/colemickens/niche";
            description = "a self-service nix binary cache tool that manages your signing key and wraps nix build to upload build products";
            license = licenses.mit;
            maintainers = with maintainers; [ colemickens ];
            platforms = platforms.linux;
          };
        };
    in rec {
      devShell = forAllSystems (system:
        pkgs_.nixpkgs.${system}.mkShell {
          name = "niche-devshell";
          nativeBuildInputs = (with pkgs_.nixpkgs.${system}; [
            cachix sops go gotools gopls pkgconfig
            bash cacert curl git jq
            nettools openssh ripgrep
          ]);
        }
      );
      packages = forAllSystems (sys: {
        niche = pkgs_.nixpkgs.${sys}.callPackage nichePkg {};
      });
      overlay = final: prev: {
        niche = prev.callPackage nichePkg {};
      };
      allPkgs = forAllSystems (sys: import inputs.nixpkgs {
        system = sys;
        config = { allowUnfree = true; };
        overlays = [ inputs.self.overlay ];
      });
      defaultPackage = forAllSystems (sys:
        inputs.self.packages.${sys}.niche
      );
    };
}

