# Taskherald

This project provides a web interface and notification service for Taskwarrior tasks with a user-defined attribute (UDA) for notification dates. It is designed for single-user, local use, and supports sending notifications via ntfy.sh.

## Features
- Polls Taskwarrior for tasks with `notification_date` UDA set
- Sends notifications via ntfy.sh
- Web UI to view/manage tasks and configuration
- Hot-reloadable config.yaml and web UI config
- Designed to run as a user-level systemd service (Linux only)

## Configuration
Edit `config.yaml` to set polling interval, notification service, and web server options. Supported services:

- `ntfy` (built-in ntfy.sh support)
- `shoutrrr` (any endpoint supported by the Shoutrrr library)

Example for ntfy.sh:
```yaml
poll_interval: 30s
notification_service: ntfy
ntfy:
	url: "https://ntfy.sh"
	topic: "task-herald"
	token: ""
web:
	listen: "127.0.0.1:8080"
	auth: false
```
Example for Shoutrrr:
```yaml
poll_interval: 30s
notification_service: shoutrrr
shoutrrr:
	urls:
		- "ntfy://task-herald?token=YOUR_TOKEN"
		- "pushover://USER:API_TOKEN"
web:
	listen: "127.0.0.1:8080"
	auth: false
```

```yaml
poll_interval: 30s
notification_service: ntfy
ntfy:
	url: "https://ntfy.sh"
	topic: "task-herald"
	token: ""
web:
	listen: "127.0.0.1:8080"
	auth: false
```

## Usage
1. Build the Go binary
2. Edit `config.yaml`
3. Run the binary as a user-level systemd service

## Taskwarrior UDA
Add the following to your `.taskrc`:

```
uda.notification_date.type=date
uda.notification_date.label=Notify At
```

Set `notification_date` on tasks to schedule notifications.

## Roadmap
- Add Home Assistant and other notification services
- Customizable notification messages based on tags/priority
- More web UI features
# Taskwarrior Notifications

This project provides a web interface and notification service for Taskwarrior tasks with a user-defined attribute (UDA) for notification dates. It is designed for single-user, local use, and supports sending notifications via ntfy.sh.

## Features
- Polls Taskwarrior for tasks with `notification_date` UDA set
- Sends notifications via ntfy.sh
- Web UI to view/manage tasks and configuration
- Hot-reloadable config.yaml and web UI config
- Designed to run as a user-level systemd service (Linux only)

## Configuration
Edit `config.yaml` to set polling interval, ntfy.sh settings, and web server options. Example:

```yaml
poll_interval: 30s
notification_service: ntfy
ntfy:
	url: "https://ntfy.sh"
	topic: "task-herald"
	token: ""
web:
	listen: "127.0.0.1:8080"
	auth: false
```

## Usage
1. Build the Go binary
2. Edit `config.yaml`
3. Run the binary as a user-level systemd service

## Taskwarrior UDA
Add the following to your `.taskrc`:

```
uda.notification_date.type=date
uda.notification_date.label=Notify At
```

Set `notification_date` on tasks to schedule notifications.

## Roadmap
- Add Home Assistant and other notification services
- Customizable notification messages based on tags/priority
- More web UI features
