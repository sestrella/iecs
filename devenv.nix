{ pkgs, ... }:

{
  packages = [
    pkgs.aws-vault
    pkgs.awscli2
    pkgs.cobra-cli
    pkgs.ssm-session-manager-plugin
  ];

  languages.go.enable = true;

  pre-commit.hooks.nixpkgs-fmt.enable = true;

  # See full reference at https://devenv.sh/reference/options/
}
