{
  inputs = {
    flake-utils.url = "github:numtide/flake-utils";
    iecs.url = "github:sestrella/iecs/nix_templates";
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
  };

  outputs = { flake-utils, iecs, nixpkgs, ... }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        iecsPkgs = iecs.packages.${system};
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        devShells.default = pkgs.mkShell {
          buildInputs = [ iecsPkgs.default ];
        };
      });
}
