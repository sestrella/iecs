{
  description = "Interactive CLI for ECS";

  inputs = {
    flake-utils.url = "github:numtide/flake-utils";
    gomod2nix.url = "github:nix-community/gomod2nix";
    gomod2nix.inputs.flake-utils.follows = "flake-utils";
    gomod2nix.inputs.nixpkgs.follows = "nixpkgs";
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
  };

  outputs = { flake-utils, gomod2nix, nixpkgs, ... }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs {
          inherit system;
          overlays = [
            gomod2nix.overlays.default
          ];
        };
      in
      {
        packages.default = pkgs.buildGoApplication {
          pname = "iecs";
          version = "0.1.0";
          src = ./.;
          modules = ./gomod2nix.toml;
        };
      }
    );
}
