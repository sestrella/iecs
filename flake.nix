{
  description = "Interactive CLI for ECS";

  inputs = {
    flake-parts.url = "github:hercules-ci/flake-parts";
    gomod2nix.url = "github:nix-community/gomod2nix";
    gomod2nix.inputs.nixpkgs.follows = "nixpkgs";
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    systems.url = "github:nix-systems/default";
  };

  outputs = inputs@{ flake-parts, gomod2nix, nixpkgs, systems, ... }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      imports = [
        flake-parts.flakeModules.easyOverlay
      ];

      systems = import systems;

      perSystem = { system, pkgs, ... }: {
        _module.args.pkgs = import nixpkgs {
          inherit system;
          overlays = [ gomod2nix.overlays.default ];
        };

        packages.default = pkgs.buildGoApplication {
          pname = "iecs";
          version = "0.1.0";
          src = ./.;
          modules = ./gomod2nix.toml;
        };
      };
    };
}
