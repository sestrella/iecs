{
  description = "Interactive CLI for ECS";

  inputs = {
    gomod2nix = {
      url = "github:nix-community/gomod2nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
    nix-filter.url = "github:numtide/nix-filter";
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    systems.url = "github:nix-systems/default";
  };

  outputs =
    { gomod2nix
    , nix-filter
    , nixpkgs
    , self
    , systems
    , ...
    }:

    {
      packages = nixpkgs.lib.genAttrs (import systems) (
        system:
        let
          pkgs = import nixpkgs {
            inherit system;
            overlays = [
              gomod2nix.overlays.default
              nix-filter.overlays.default
            ];
          };
        in
        {
          default = pkgs.callPackage ./default.nix { };
          demo = pkgs.callPackage ./default.nix {
            pname = "iecs-demo";
            tags = [ "DEMO" ];
          };
        }
      );

      overlays.default = final: prev: {
        iecs = self.packages.${prev.system}.default;
      };

      templates = {
        default = {
          description = "Shows how to install iecs in flake";
          path = ./templates/default;
        };
        devenv = {
          description = "Shows how to install iecs in devenv";
          path = ./templates/devenv;
        };
        overlay = {
          description = "Shows how to use iecs as an overlay";
          path = ./templates/overlay;
        };
      };
    };
}
