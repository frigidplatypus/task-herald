go test ./...
# Task Herald

Task Herald watches Taskwarrior tasks and sends scheduled notifications via ntfy for tasks with a notification UDA.

Overview
- Polls Taskwarrior and sends ntfy notifications for tasks with a configured UDA.
- Provides a Home Manager module (flake) for deployment.
- Optional HTTP API for health and lightweight task management.

Quick start

Build locally:

```sh
git clone https://github.com/frigidplatypus/task-herald.git
cd task-herald
go build -o task-herald ./cmd
```

Run with a config file:

```sh
./task-herald --config ./config.yaml
```

Configuration (`config.yaml`)

Minimal example (trim to the fields you need):

```yaml
log_level: info
poll_interval: 30s
sync_interval: 5m

ntfy:
  url: "https://ntfy.sh"
  topic: "your-topic"
  # or use a file: topic_file: "/run/secrets/ntfy-topic"
  token: "" # or prefer a file-backed secret
  headers:
    X-Title: "{{.Project}}"
    X-Default: "{{.Priority}}"
  actions_enabled: true

notification_message: "" # optional Go template

udas:
  notification_date: notification_date
  repeat_enable: notification_repeat_enable
  repeat_delay: notification_repeat_delay

http:
  # Prefer configuring host and port separately; `addr` remains for backward compatibility
  host: "127.0.0.1"
  port: 43000
  addr: "127.0.0.1:43000" # optional, deprecated: prefer host+port
  debug: false

domain: "example.local" # public-facing domain used to build acknowledgement URLs in notifications
```

Home Manager module (flake)

The flake exports a Home Manager module at `homeManagerModules.task-herald`. Import it into your Home Manager configuration and set options under `services.task-herald.settings`.

Example (Nix snippet):

```nix
# Example flake that uses the task-herald Home Manager module (copy into
# e.g. `flake.nix` in your dotfiles repo). This creates `homeConfigurations`
# for the listed systems. Adjust `home.username` and secret paths to taste.
{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-24.11";
    home-manager.url = "github:nix-community/home-manager";
    task-herald.url = "github:frigidplatypus/task-herald";
  };

  outputs = { self, nixpkgs, home-manager, task-herald, ... }:
    let
      systems = [ "x86_64-linux" ];
    in
    {
      homeConfigurations = builtins.listToAttrs (map (system:
        let
          pkgs = import nixpkgs { inherit system; };
        in
        {
          name = system;
          value = home-manager.lib.homeManagerConfiguration {
            inherit pkgs;
            modules = [ task-herald.homeManagerModules.task-herald ];
            configuration = {
              home.username = "alice";
              home.homeDirectory = "/home/alice";

              services.task-herald.enable = true;
              services.task-herald.settings = {
                poll_interval = "30s";
                sync_interval = "5m";
                ntfy = { url = "https://ntfy.sh"; topic_file = "/run/secrets/ntfy-topic"; };
                http = {
                  addr = "127.0.0.1:43000";
                  auth_token_file = "/run/secrets/task-herald-token";
                };
              };
            };
          };
        }
      ) systems);
    };
}
```

Key Home Manager options (under `settings`):
- `ntfy.url`, `ntfy.topic` or `ntfy.topic_file`
- `ntfy.token` or use Home Manager file options
- `poll_interval`, `sync_interval`, `log_level`
- `http.addr` (bind address)
- `http.tls_cert`, `http.tls_key` (inline paths)
- `http.tls_cert_file`, `http.tls_key_file` (file-backed TLS paths)
- `http.auth_token`, `http.auth_token_file` (inline or file-backed bearer token)
- `http.debug` (enable `/api/debug`)

Security note: prefer `*_file` options for secrets and keep secret files restrictive (e.g., 0600).

HTTP API (optional)

When enabled, the API exposes these endpoints. If `http.auth_token` or `http.auth_token_file` is set, include `Authorization: Bearer <token>`.

- GET /api/health
  - Response: 200 OK, `{ "status": "ok" }`

- POST /api/create-task
  - Request: JSON
    - `description` (string, required)
    - `project` (string, optional)
    - `tags` ([]string, optional)
    - `annotations` ([]string, optional)
    - `notification_date` (string, optional)
  - Response: 201 Created, `{ "uuid": "<task-uuid>", "message": "created" }`

- POST /api/acknowledge
  - Request JSON: `uuid` (required), `repeat_delay` (optional)
  - Response: 200 OK, `{ "acknowledged": true }`

- GET /api/debug
  - Response: 200 OK, `{ "debug": "ok" }` (enabled with `http.debug`)

Taskwarrior UDA setup

Add these lines to your `~/.taskrc` to enable notification UDAs:

```text
uda.notification_date.type=date
uda.notification_date.label=Notify At

uda.notification_repeat_enable.type=string
uda.notification_repeat_enable.values=true,false
uda.notification_repeat_enable.label=Repeat Notification

uda.notification_repeat_delay.type=duration
uda.notification_repeat_delay.label=Repeat Delay
```

Development

Run tests:

```sh
go test ./...
```

Contributions welcome — open issues or PRs.
        };
