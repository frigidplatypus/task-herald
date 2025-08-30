# Task Herald

Task Herald provides a notification service for Taskwarrior tasks with scheduled notification dates. It monitors your Taskwarrior tasks and sends notifications when tasks with a `notification_date` user-defined attribute (UDA) are due.

## Features

- **Flexible Notifications**: Supports any notification service via [Shoutrrr](https://containrrr.dev/shoutrrr/) (ntfy, Discord, Slack, email, etc.)
- **Taskwarrior Integration**: Polls Taskwarrior for tasks with `notification_date` UDA
- **Web Interface**: View and manage tasks through a web UI
- **Credential Security**: Support for reading notification URLs from files (for secure credential management)
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

      web = {
        listen = "127.0.0.1:8080";
        auth = false;
      };

      ntfy = {
        url = "https://ntfy.sh";
        topic = "QWvwi17Z";
        token = "";
        headers = {
          X-Title = "{{.Project}}";
          X-Default = "{{.Priority}}";
          # X-Click = "https://example.com/task/{{.UUID}}";
          # X-Actions = ''[{"action":"view","label":"View Task","url":"https://example.com/task/{{.UUID}}"}]'';
        };
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

# Web server settings
web:
  listen: "127.0.0.1:8080"       # Address and port to listen on
  auth: false                    # Enable authentication (true/false)

# ntfy notification settings
ntfy:
  url: "https://ntfy.sh"
  topic: "QWvwi17Z"
  token: ""
  headers:
    X-Title: "{{.Project}}"
    X-Default: "{{.Priority}}"
    # X-Click: "https://example.com/task/{{.UUID}}"
    # X-Actions: '[{"action":"view","label":"View Task","url":"https://example.com/task/{{.UUID}}"}]'

# Custom notification message (Go template, see TaskInfo struct for fields)
# notification_message: "🔔 {{.Description}} (Due: {{.Due}})"
# notification_message: ""

# UDA field mapping for notification features
udas:
  notification_date: notification_date
  repeat_enable: notification_repeat_enable
  repeat_delay: notification_repeat_delay
```

### Supported Notification Services

Task Herald uses [Shoutrrr](https://containrrr.dev/shoutrrr/) for notifications, supporting:

- **ntfy**: `ntfy://[username:password@]ntfy.sh/topic`
- **Discord**: `discord://token@channel`
- **Slack**: `slack://bottoken@channel`
- **Email**: `smtp://username:password@host:port/?from=fromAddress&to=recipient`
- **Many others**: See [Shoutrrr documentation](https://containrrr.dev/shoutrrr/v0.8/services/)

### Credential Security

For security, you can store notification URLs in files instead of configuration:

```yaml
# Instead of exposing credentials in config
# shoutrrr_url: "discord://secret-token@channel"

# Read from file
shoutrrr_url_file: "/run/secrets/discord-webhook"
```

This is especially useful with:
- systemd credentials: `LoadCredential=discord-webhook:/path/to/webhook`
- Docker secrets
- Kubernetes secrets
- Home Manager secrets management


## Taskwarrior Setup

Add the notification UDA and nagging (repeat) UDAs to your `.taskrc`:

```bash
# Add to ~/.taskrc
uda.notification_date.type=date
uda.notification_date.label=Notify At

# Enable repeating (nagging) notifications per task
uda.taskherald.repeat_enable.type=string
uda.taskherald.repeat_enable.values=true,false
uda.taskherald.repeat_enable.label=Repeat Notification

# Set the nagging interval (duration)
uda.taskherald.repeat_delay.type=duration
uda.taskherald.repeat_delay.label=Repeat Delay
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
