{
  description = "task-herald flake: builds the task-herald binary and exports NixOS and Home Manager modules";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-24.11";
    flake-utils.url = "github:numtide/flake-utils";
    home-manager.url = "github:nix-community/home-manager";
  };

  outputs = { self, nixpkgs, flake-utils, home-manager, ... }:
    let
      homeManagerModule = { config, pkgs, lib, ... }:
        let
          taskHeraldPkg = self.packages.${pkgs.system}.default;
        in
        {
          options.services.task-herald = {
            enable = lib.mkOption {
              type = lib.types.bool;
              default = false;
              description = "Enable task-herald user service";
            };
            configFile = lib.mkOption {
              type = lib.types.str;
              default = ".config/task-herald/config.yaml";
              description = "Path to the config file relative to home directory";
            };
            configText = lib.mkOption {
              type = lib.types.nullOr lib.types.str;
              default = null;
              description = "Config file contents for the user service (overrides settings)";
            };
            package = lib.mkOption {
              type = lib.types.package;
              default = taskHeraldPkg;
              description = "Package to run";
            };
            settings = lib.mkOption {
              type = lib.types.submodule {
                options = {
                  poll_interval = lib.mkOption {
                    type = lib.types.str;
                    default = "30s";
                    description = "How often to poll Taskwarrior for tasks";
                  };
                  sync_interval = lib.mkOption {
                    type = lib.types.str;
                    default = "5m";
                    description = "How often to run 'task sync'";
                  };
                  log_level = lib.mkOption {
                    type = lib.types.str;
                    default = "info";
                    description = "Logging level: error, warn, info, debug, verbose";
                  };
                  shoutrrr_url = lib.mkOption {
                    type = lib.types.str;
                    default = "";
                    description = "Shoutrrr URL for notifications";
                  };
                  shoutrrr_url_file = lib.mkOption {
                    type = lib.types.nullOr lib.types.str;
                    default = null;
                    description = "Path to file containing shoutrrr URL (alternative to shoutrrr_url)";
                  };
                  notification_message = lib.mkOption {
                    type = lib.types.nullOr lib.types.str;
                    default = null;
                    description = "Custom notification message template";
                  };
                  web = lib.mkOption {
                    type = lib.types.submodule {
                      options = {
                        host = lib.mkOption {
                          type = lib.types.str;
                          default = "127.0.0.1";
                          description = "Host address to listen on";
                        };
                        port = lib.mkOption {
                          type = lib.types.port;
                          default = 8080;
                          description = "Port to listen on";
                        };
                        auth = lib.mkOption {
                          type = lib.types.bool;
                          default = false;
                          description = "Enable authentication";
                        };
                      };
                    };
                    default = {};
                    description = "Web server settings";
                  };
                };
              };
              default = {};
              description = "Task-herald runtime settings that will be rendered into config.yaml";
            };
          };
          config = lib.mkIf config.services.task-herald.enable {
            assertions = [
              {
                assertion = (config.services.task-herald.settings.shoutrrr_url != "") || (config.services.task-herald.settings.shoutrrr_url_file != null);
                message = "Either services.task-herald.settings.shoutrrr_url or services.task-herald.settings.shoutrrr_url_file must be set when task-herald is enabled";
              }
            ];
            home.packages = [ taskHeraldPkg ];
            home.file.${config.services.task-herald.configFile} = {
              text = if config.services.task-herald.configText != null
                     then config.services.task-herald.configText
                     else pkgs.lib.generators.toYAML {} (
                       lib.filterAttrs (n: v: v != null) config.services.task-herald.settings
                     );
            };
            systemd.user.services."task-herald" = {
              Unit = {
                Description = "Task Herald (user service)";
              };
              Install = {
                WantedBy = [ "default.target" ];
              };
              Service = {
                ExecStart = "${taskHeraldPkg}/bin/task-herald --config %h/${config.services.task-herald.configFile}";
                Restart = "always";
                WorkingDirectory = "%h/.local/state/task-herald";
                StateDirectory = "task-herald";
                Environment = "PATH=${pkgs.taskwarrior}/bin:${pkgs.coreutils}/bin:/run/wrappers/bin:/usr/bin:/bin";
              };
            };
          };
        };
    in
    (flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };

        taskHeraldPkg = pkgs.stdenv.mkDerivation {
          pname = "task-herald";
          version = "0.0.0";
          src = ./.;
          nativeBuildInputs = [ pkgs.go ];
          buildPhase = ''
            export GOPRIVATE=github.com/frigidplatypus
            export GOCACHE="$PWD/.gocache"
            export GOPATH="$PWD/.gopath"
            mkdir -p "$GOCACHE" "$GOPATH"
            mkdir -p $out
            cd $src
            PATH=$PATH:${pkgs.go}/bin
            GOFLAGS="-mod=vendor" go build -ldflags "-s -w" -o $out/bin/task-herald ./cmd
            mkdir -p $out/web/templates
            mkdir -p $out/web/static
            cp -r web/templates/* $out/web/templates/
            cp -r web/static/* $out/web/static/
          '';
          installPhase = ''
            # Remove all sources except the binary and assets for a minimal output
            find $out -mindepth 1 -maxdepth 1 ! -name bin ! -name web -exec rm -rf {} +
          '';
          meta = {
            description = "Taskwarrior notifications service";
            license = "MIT";
          };
        };
      in
      {
        packages.default = taskHeraldPkg;
        apps.default = {
          type = "app";
          program = "${taskHeraldPkg}/bin/task-herald";
          meta = {
            description = "Taskwarrior notifications service";
          };
        };
      }
    )) // {
      homeManagerModules = { "task-herald" = homeManagerModule; };
    };
}
