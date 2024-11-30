{ pkgs, ... }:

{
  packages = [
    pkgs.aws-vault
    pkgs.awscli2
    pkgs.cobra-cli
    pkgs.gomod2nix
    pkgs.ssm-session-manager-plugin
  ];

  languages.go.enable = true;

  pre-commit.hooks = {
    gofmt.enable = true;
    golangci-lint.enable = true;
    gomod2nix = {
      enable = true;
      entry = "${pkgs.gomod2nix}/bin/gomod2nix";
      pass_filenames = false;
      files = "go.mod";
    };
    gotest.enable = true;
    nixpkgs-fmt.enable = true;
  };

  # See full reference at https://devenv.sh/reference/options/
}
