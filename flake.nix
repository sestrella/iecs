{
  description = "Interactive CLI for ECS";

  inputs = {
    flake-parts.url = "github:hercules-ci/flake-parts";
    gomod2nix.url = "github:nix-community/gomod2nix";
    gomod2nix.inputs.nixpkgs.follows = "nixpkgs";
    nix-filter.url = "github:numtide/nix-filter";
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    systems.url = "github:nix-systems/default";
  };

  outputs = inputs@{ flake-parts, gomod2nix, nix-filter, nixpkgs, systems, ... }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      imports = [
        flake-parts.flakeModules.easyOverlay
      ];

      systems = import systems;

      perSystem = { self', pkgs, system, ... }: {
        _module.args.pkgs = import nixpkgs {
          inherit system;
          overlays = [ gomod2nix.overlays.default ];
        };

        packages.default = pkgs.buildGoApplication {
          pname = "iecs";
          version = "0.1.0";
          src = nix-filter.lib {
            root = ./.;
            include = [
              "./go.mod"
              "./go.sum"
              "./main.go"
              "cmd"
              "selector"
            ];
          };
          modules = ./gomod2nix.toml;
        };

        overlayAttrs = {
          iecs = self'.packages.default;
        };
      };
    };
}
