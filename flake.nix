{
  description = "niche";

  inputs = {
    nixpkgs = { url = "github:colemickens/nixpkgs/nixos-unstable"; };
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

      mkSystem = sys: pkgs_: hostname:
        pkgs_.lib.nixosSystem {
          system = sys;
          modules = [(./. + "/hosts/${hostname}/configuration.nix")];
          specialArgs = { inherit inputs; };
        };
    in rec {
      defaultPackage = forAllSystems (sys: import inputs.nixpkgs {
        # whatever to build the go app here
      });
    };
}

