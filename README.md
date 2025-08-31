# Task Herald


Task Herald provides a notification service for Taskwarrior tasks with scheduled notification dates. It monitors your Taskwarrior tasks and sends notifications when tasks with a `notification_date` user-defined attribute (UDA) are due, using [ntfy](https://ntfy.sh/) for push notifications.

## Features

- **ntfy Notifications**: Sends push notifications via [ntfy](https://ntfy.sh/)
- **Taskwarrior Integration**: Polls Taskwarrior for tasks with `notification_date` UDA
- **Nix Integration**: First-class Nix flake with home-manager module
- **Systemd Service**: Designed to run as a user-level systemd service

## Installation

### Using Nix Flake (Recommended)

Add this flake as an input to your system flake:

```nix
{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    home-manager.url = "github:nix-community/home-manager";
    task-herald.url = "github:frigidplatypus/task-herald";
  };
}
```

### Using Home Manager

Import the home-manager module and configure:

```nix
{
  imports = [
    inputs.task-herald.homeManagerModules.task-herald
  ];

  services.task-herald = {
    enable = true;
    settings = {
      poll_interval = "30s";
      sync_interval = "5m";
      log_level = "verbose";




      ntfy = {
        url = "https://ntfy.sh";
        topic = "QWvwi17Z"; # Or use topic_file below
        # topic_file = "/run/secrets/ntfy-topic"; # Alternative: read topic from file
        token = "";
        headers = {
          X-Title = "{{.Project}}";
          X-Default = "{{.Priority}}";
        };
      };

      # HTTP API (optional)
      # If you enable the HTTP API, task-herald will start a small HTTP server
      # serving a health endpoint and management endpoints. You can configure
      # address, TLS cert/key, and an authentication token.
      http = {
        addr = "127.0.0.1:43000"; # or ":0" for ephemeral port
        tls_cert = null; # path to TLS certificate file (optional)
        tls_key = null;  # path to TLS private key file (optional)
  # You can provide secrets inline (auth_token, tls_cert, tls_key) or
  # point to files containing the secret values. Using file-backed
  # options is recommended when storing configuration in a public repo.
  auth_token = null; # Bearer token to require for API requests (optional)
  auth_token_file = null; # Path to file containing bearer token (optional)
  tls_cert_file = null; # Path to TLS certificate file (optional, file-backed)
  tls_key_file = null;  # Path to TLS private key file (optional, file-backed)
        debug = false; # enable /api/debug endpoint
      };

      # Custom notification message template
      # notification_message = "🔔 {{.Description}} (Due: {{.Due}})";

      udas = {
        notification_date = "notification_date";
        repeat_enable = "notification_repeat_enable";
        repeat_delay = "notification_repeat_delay";
      };
    };
  };
}
```

### Manual Installation

1. **Build from source:**
   ```bash
   git clone https://github.com/frigidplatypus/task-herald.git
   cd task-herald
   go build -o task-herald ./cmd
   ```

2. **Create configuration:**
   ```bash
   cp config.example.yaml config.yaml
   # Edit config.yaml with your settings
   ```

3. **Run the service:**
   ```bash
   ./task-herald --config config.yaml
   ```

## Configuration

### Configuration File Example

```yaml
# Example config for task-herald

# Logging level: error, warn, info, debug, verbose
log_level: verbose

# How often to poll Taskwarrior for tasks (e.g., 30s, 1m)
poll_interval: 30s

# How often to run 'task sync' (e.g., 5m, 10m)
sync_interval: 5m




# ntfy notification settings
ntfy:
  url: "https://ntfy.sh"
  topic: "QWvwi17Z"            # Or use topic_file below
  # topic_file: "/run/secrets/ntfy-topic"   # Alternative: read topic from file
  token: ""
  headers:
    X-Title: "{{.Project}}"
    X-Default: "{{.Priority}}"
  actions_enabled: true

# Custom notification message (Go template, see TaskInfo struct for fields)
# notification_message: "🔔 {{.Description}} (Due: {{.Due}})"
# notification_message: ""

# UDA field mapping for notification features
udas:
  notification_date: notification_date
  repeat_enable: notification_repeat_enable
  repeat_delay: notification_repeat_delay
```


### ntfy Notification Service

Task Herald uses [ntfy](https://ntfy.sh/) for notifications. You can configure the ntfy server, topic, token, and headers in your config file. For security, you can store the topic in a file using the `topic_file` option.


## Taskwarrior Setup

Add the notification UDA and nagging (repeat) UDAs to your `.taskrc`:

```bash
# Add to ~/.taskrc
uda.notification_date.type=date
uda.notification_date.label=Notify At

# Enable repeating (nagging) notifications per task
uda.notification_repeat_enable.type=string
uda.notification_repeat_enable.values=true,false
uda.notification_repeat_enable.label=Repeat Notification

# Set the nagging interval (duration)
uda.notification_repeat_delay.type=duration
uda.notification_repeat_delay.label=Repeat Delay
```

### About Taskwarrior Durations

Taskwarrior supports a flexible [duration format](https://taskwarrior.org/docs/durations/) for UDA values of type `duration`. Examples:

- `10m` (10 minutes)
- `1h` (1 hour)
- `2d` (2 days)
- `1w` (1 week)
- `1h30m` (1 hour, 30 minutes)

You can use these values for `taskherald.repeat_delay` to control how often a nagging notification is sent for a task.

### Example: Setting up a nagging notification

```bash
# Add a task that will nag every 30 minutes until acknowledged
task add "Take a break" notification_date:now+1min taskherald.repeat_enable:true taskherald.repeat_delay:30m
```

Set notification dates on tasks:

```bash
# Set notification for specific date/time
task add "Important meeting" notification_date:2024-03-15T14:30

# Set notification relative to now
task add "Call dentist" notification_date:tomorrow

# Modify existing task
task 1 modify notification_date:2024-03-20T09:00
```

## Command Line Options

```bash
task-herald --config /path/to/config.yaml
```

Configuration precedence (highest to lowest):
1. `--config` CLI flag
2. `TASK_HERALD_CONFIG` environment variable
3. `./config.yaml` (current directory)
4. `/var/lib/task-herald/config.yaml` (system location)

## Development

### Using devenv (Nix)

```bash
git clone https://github.com/frigidplatypus/task-herald.git
cd task-herald
nix develop
```

### Using Go directly

```bash
git clone https://github.com/frigidplatypus/task-herald.git
cd task-herald
go mod download
go run ./cmd --config config.yaml
```
