{ config
, lib
, pkgs
, ...
}:

{
  env.GEMINI_API_KEY = config.secretspec.secrets.GEMINI_API_KEY or "";

  packages = [
    pkgs.asciinema
    pkgs.asciinema-agg
    pkgs.aws-vault
    pkgs.awscli2
    pkgs.cobra-cli
    pkgs.gemini-cli
    pkgs.ssm-session-manager-plugin
  ];

  languages.go.enable = true;

  git-hooks.hooks = {
    auto-commit-msg = {
      enable = true;
      entry = lib.getExe pkgs.auto-commit-msg;
      stages = [ "prepare-commit-msg" ];
      verbose = true;
    };
    golangci-lint.enable = true;
    golines.enable = true;
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
