{
  pkgs,
  lib,
  config,
  inputs,
  ...
}:

{
  # https://devenv.sh/basics/

  # https://devenv.sh/packages/
  packages = [
    pkgs.git
    pkgs.reflex
    pkgs.just
    pkgs.nodePackages.concurrently
  ];
  languages.javascript = {
    enable = true;
    package = pkgs.nodejs_20;
  };

  # dotenv.enable = true;

  # Expose repo-local testing environment variables via devenv
  # These mirror the .env values used for local testing and make them
  # available whenever you `devenv enter` or direnv evaluates the shell.

  languages.go = {
    enable = true;
    enableHardeningWorkaround = true;
  };

  # https://devenv.sh/languages/
  # languages.rust.enable = true;

  # https://devenv.sh/processes/
  # processes.cargo-watch.exec = "cargo-watch";

  # https://devenv.sh/services/
  # services.postgres.enable = true;

  # https://devenv.sh/scripts/

  # https://devenv.sh/tasks/
  # tasks = {
  #   "myproj:setup".exec = "mytool build";
  #   "devenv:enterShell".after = [ "myproj:setup" ];
  # };

  # https://devenv.sh/tests/
  enterTest = ''
    echo "Running tests"
    git --version | grep --color=auto "${pkgs.git.version}"
  '';

  # https://devenv.sh/git-hooks/
  # git-hooks.hooks.shellcheck.enable = true;

  # See full reference at https://devenv.sh/reference/options/
}
