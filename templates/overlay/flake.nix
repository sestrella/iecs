{
  inputs = {
    iecs.url = "github:sestrella/iecs";
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    systems.url = "github:nix-systems/default";
  };

  nixConfig = {
    extra-substituters = "https://sestrella.cachix.org";
    extra-trusted-public-keys = "sestrella.cachix.org-1:uf75o4yckcsAOFu6ldfPug/kTUMybvT0IY61sck2qnA=";
  };

  outputs = { iecs, nixpkgs, systems, ... }:
    let
      eachSystem = nixpkgs.lib.genAttrs (import systems);
    in
    {
      devShells = eachSystem (system:
        let
          pkgs = import nixpkgs {
            inherit system;
            overlays = [ iecs.overlays.default ];
          };
        in
        {
          default = pkgs.mkShell {
            buildInputs = [ pkgs.iecs ];
          };
        });
    };
}
