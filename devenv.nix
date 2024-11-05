{ pkgs, lib, ... }:

{
  packages = [
    pkgs.aws-vault
    pkgs.awscli2
    pkgs.cobra-cli
    pkgs.ssm-session-manager-plugin
  ] ++ lib.optionals pkgs.stdenv.isDarwin [
    pkgs.darwin.Security
  ];

  languages.go.enable = true;

  languages.rust.enable = true;
  languages.rust.channel = "stable";

  pre-commit.hooks.clippy.enable = true;
  pre-commit.hooks.nixpkgs-fmt.enable = true;
  pre-commit.hooks.rustfmt.enable = true;

  # See full reference at https://devenv.sh/reference/options/
}
