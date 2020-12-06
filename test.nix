let
  nixpkgs = builtins.fetchTarball { url = "https://github.com/nixos/nixpkgs/archive/master.tar.gz"; };
  pkgs = import nixpkgs {
    overlays = [
      (import (builtins.fetchTarball { url="https://github.com/colemickens/nixpkgs-wayland/archive/master.tar.gz"; }))
    ];
  };
in
  pkgs
