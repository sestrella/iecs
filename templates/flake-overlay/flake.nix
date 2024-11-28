{
  inputs = {
    iecs.url = "github:sestrella/iecs/nix_templates";
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    systems.url = "github:nix-systems/default";
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
