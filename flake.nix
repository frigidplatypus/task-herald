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
                  ntfy = lib.mkOption {
                    type = lib.types.submodule {
                      options = {
                        url = lib.mkOption {
                          type = lib.types.str;
                          default = "https://ntfy.sh";
                          description = "ntfy server URL";
                        };
                        topic = lib.mkOption {
                          type = lib.types.str;
                          default = "";
                          description = "ntfy topic name";
                        };
                        topic_file = lib.mkOption {
                          type = lib.types.nullOr lib.types.str;
                          default = null;
                          description = "Path to file containing ntfy topic (alternative to topic)";
                        };
                        token = lib.mkOption {
                          type = lib.types.str;
                          default = "";
                          description = "ntfy API token (optional)";
                        };
                        headers = lib.mkOption {
                          type = lib.types.attrsOf lib.types.str;
                          default = {
                            X-Title = "{{.Project}}";
                            X-Default = "{{.Priority}}";
                          };
                          description = "ntfy notification headers (Go template values allowed)";
                        };
                        actions_enabled = lib.mkOption {
                          type = lib.types.bool;
                          default = false;
                          description = "Automatically add X-Actions delay button to notifications";
                        };
                      };
                    };
                    default = {};
                    description = "ntfy notification settings";
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
                          type = lib.types.int;
                          default = 8080;
                          description = "Port to listen on";
                        };
                        domain = lib.mkOption {
                          type = lib.types.str;
                          default = "localhost";
                          description = "Domain or hostname for web UI (used for X-Actions URLs)";
                        };
                      };
                    };
                    default = {};
                    description = "Web server settings";
                  };
                  udas = lib.mkOption {
                    type = lib.types.submodule {
                      options = {
                        notification_date = lib.mkOption {
                          type = lib.types.str;
                          default = "notification_date";
                          description = "UDA field for notification date";
                        };
                        repeat_enable = lib.mkOption {
                          type = lib.types.str;
                          default = "notification_repeat_enable";
                          description = "UDA field for enabling repeat notifications";
                        };
                        repeat_delay = lib.mkOption {
                          type = lib.types.str;
                          default = "notification_repeat_delay";
                          description = "UDA field for repeat delay duration";
                        };
                      };
                    };
                    default = {};
                    description = "UDA field mapping for notification features";
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
                assertion = (config.services.task-herald.settings.ntfy.url != null && config.services.task-herald.settings.ntfy.url != "")
                  && (
                    (config.services.task-herald.settings.ntfy.topic != null && config.services.task-herald.settings.ntfy.topic != "")
                    || (config.services.task-herald.settings.ntfy.topic_file != null && config.services.task-herald.settings.ntfy.topic_file != "")
                  );
                message = "services.task-herald.settings.ntfy.url and (ntfy.topic or ntfy.topic_file) must be set when task-herald is enabled";
              }
            ];
            home.packages = [ taskHeraldPkg ];
            home.file.${config.services.task-herald.configFile} =
              if config.services.task-herald.configText != null then {
                text = config.services.task-herald.configText;
              } else {
                source = (pkgs.formats.yaml {}).generate "config.yaml" (
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
                Environment = [ "TASK_HERALD_ASSET_DIR=${taskHeraldPkg}/share/task-herald/web" ];
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
            go build -ldflags "-s -w" -o $out/bin/task-herald ./cmd
          '';
          installPhase = ''
            mkdir -p $out/bin
            cp bin/task-herald $out/bin/
            mkdir -p $out/share/task-herald/web/templates
            mkdir -p $out/share/task-herald/web/static
            if [ -d web/templates ]; then cp -r web/templates/* $out/share/task-herald/web/templates/; fi
            if [ -d web/static ]; then cp -r web/static/* $out/share/task-herald/web/static/; fi
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
