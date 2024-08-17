{ pkgs, ... }:

{
  packages = [
    pkgs.cobra-cli
  ];

  languages.go.enable = true;

  # See full reference at https://devenv.sh/reference/options/
}
