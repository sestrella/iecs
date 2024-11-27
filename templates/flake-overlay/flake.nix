{
  inputs = {
    flake-utils.url = "github:numtide/flake-utils";
    iecs.url = "github:sestrella/iecs/nix_templates";
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
  };

  outputs = { flake-utils, iecs, nixpkgs, ... }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs {
          inherit system;
          overlays = [ iecs.overlays.default ];
        };
      in
      {
        devShells.default = pkgs.mkShell {
          buildInputs = [ pkgs.iecs ];
        };
      });
}
