{ pkgs, ... }:

{
  packages = [
    pkgs.asciinema
    pkgs.aws-vault
    pkgs.awscli2
    pkgs.claude-code
    pkgs.cobra-cli
    pkgs.gomod2nix
    pkgs.ssm-session-manager-plugin
  ];

  languages.go.enable = true;

  git-hooks.hooks = {
    golangci-lint.enable = true;
    golines.enable = true;
    gomod2nix = {
      enable = true;
      entry = "${pkgs.gomod2nix}/bin/gomod2nix";
      pass_filenames = false;
      files = "go.mod";
    };
    nixpkgs-fmt.enable = true;
  };

  # See full reference at https://devenv.sh/reference/options/
}
