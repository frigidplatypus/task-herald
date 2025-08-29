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

  taskHerald = {
    enable = true;
    settings = {
      # Required: notification service URL
      shoutrrr_url = "ntfy://task-herald@ntfy.sh";
      
      # Alternative: read URL from file (more secure for credentials)
      # shoutrrr_url_file = "/run/secrets/shoutrrr-url";
      
      # Optional settings with defaults
      poll_interval = "30s";
      sync_interval = "5m";
      log_level = "info";
      
      # Web interface settings
      web = {
        listen = "127.0.0.1:8080";
        auth = false;
      };
      
      # Custom notification message template
      notification_message = "🔔 {{.Description}} (Due: {{.Due}})";
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

### Configuration File Format

```yaml
# How often to poll Taskwarrior for tasks
poll_interval: 30s

# How often to run 'task sync' 
sync_interval: 5m

# Logging level: error, warn, info, debug, verbose
log_level: info

# Web server settings
web:
  listen: "127.0.0.1:8080"
  auth: false

# Notification service URL (see Shoutrrr documentation)
shoutrrr_url: "ntfy://task-herald@ntfy.sh"

# Alternative: read URL from file (more secure)
# shoutrrr_url_file: "/path/to/shoutrrr-url.txt"

# Custom notification message (Go template)
notification_message: "🔔 {{.Description}} (Due: {{.Due}})"
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

Add the notification UDA to your `.taskrc`:

```bash
# Add to ~/.taskrc
uda.notification_date.type=date
uda.notification_date.label=Notify At
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

## Roadmap

- [ ] Multiple notification service support per task
- [ ] Task filtering and advanced scheduling
- [ ] Enhanced web UI with task management
- [ ] Integration with more task management systems
- [ ] Mobile app notifications
