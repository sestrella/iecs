{
  description = "Interactive CLI for ECS";

  inputs = {
    gomod2nix.url = "github:nix-community/gomod2nix";
    gomod2nix.inputs.nixpkgs.follows = "nixpkgs";
    nix-filter.url = "github:numtide/nix-filter";
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    systems.url = "github:nix-systems/default";
  };

  outputs =
    { self
    , gomod2nix
    , nix-filter
    , nixpkgs
    , systems
    , ...
    }:
    let
      forAllSystems = nixpkgs.lib.genAttrs (import systems);
    in
    {
      packages = forAllSystems (
        system:
        let
          pkgs = import nixpkgs {
            inherit system;
            overlays = [ gomod2nix.overlays.default ];
          };
          mkPackage =
            { versionFunc ? version: version
            , tags ? [ ]
            ,
            }:
            pkgs.buildGoApplication {
              inherit tags;
              pname = "iecs";
              version = versionFunc (nixpkgs.lib.trim (builtins.readFile ./version.txt));
              src = nix-filter.lib {
                root = ./.;
                include = [
                  "./go.mod"
                  "./go.sum"
                  "./main.go"
                  "client"
                  "cmd"
                  "selector"
                  "version.txt"
                ];
              };
              modules = ./gomod2nix.toml;
            };
        in
        {
          default = mkPackage { };
          demo = mkPackage {
            versionFunc = version: "${version}-demo";
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
