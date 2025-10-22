{ lib
, nix-filter
, pkgs
, pname ? "iecs"
, tags ? [ ]
,
}:

pkgs.buildGoApplication {
  inherit pname tags;
  version = lib.trim (builtins.readFile ./version.txt);
  src = nix-filter {
    root = ./.;
    include = [
      "client"
      "cmd"
      "selector"
      ./go.mod
      ./go.sum
      ./main.go
      ./version.txt
    ];
  };
  modules = ./gomod2nix.toml;
}
