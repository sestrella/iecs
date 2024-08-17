{ pkgs, ... }:

{
  packages = [
    pkgs.aws-vault
    pkgs.awscli2
    pkgs.cobra-cli
  ];

  languages.go.enable = true;

  # See full reference at https://devenv.sh/reference/options/
}
