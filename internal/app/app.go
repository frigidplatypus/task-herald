package app

import (
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"task-herald/internal/config"
	"task-herald/internal/notify"
	"task-herald/internal/taskwarrior"
	"task-herald/internal/util"
	"task-herald/internal/web"
	"time"
)

func Run(configOverride string) error {
	fmt.Println("Taskwarrior Notifications service starting...")

	// Precedence: CLI override -> TASK_HERALD_CONFIG env -> ./config.yaml -> /var/lib/task-herald/config.yaml
	cfgPath := configOverride
	if cfgPath == "" {
		cfgPath = os.Getenv("TASK_HERALD_CONFIG")
	}
	if cfgPath == "" {
		if _, err := os.Stat("./config.yaml"); err == nil {
			cfgPath = "./config.yaml"
		} else {
			cfgPath = "/var/lib/task-herald/config.yaml"
		}
	}

	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	config.Set(cfg)

	// Shared state for polled tasks
	var (
		mu               sync.RWMutex
		tasks            []taskwarrior.Task
		notified         = make(map[string]struct{}) // Key: UUID|notification_date
		lastNotifiedDate = make(map[string]string)   // Key: UUID, Value: last seen notification_date
	)

	// Set up polling and syncing
	taskCh := make(chan []taskwarrior.Task)
	stopCh := make(chan struct{})
	go taskwarrior.Poller(cfg.PollInterval, taskCh, stopCh)
	go taskwarrior.SyncTaskwarrior(stopCh)

	// Update tasks on poll
	go func() {
		for t := range taskCh {
			mu.Lock()
			tasks = t
			// Log only if a task's notification date is new or changed
			now := time.Now()
			for _, task := range t {
				if task.NotificationDate == "" {
					continue
				}
				prev := lastNotifiedDate[task.UUID]
				if prev != task.NotificationDate {
					notifyAt, err := util.ParseNotificationDate(task.NotificationDate)
					if err == nil {
						// Only log if notification time is in the future
						nowLocal := now.In(time.Local)
						notifyLocal := notifyAt.In(time.Local)

						// Only show tasks with future notification dates
						if notifyLocal.After(nowLocal) {
							fmt.Printf("[task] Task %s (%s) has notification date: %s\n", task.UUID, task.Description, notifyLocal.Format("2006-01-02 15:04:05 MST"))
						}
					} else {
						fmt.Printf("[task] Task %s (%s) has notification date: %s (parse error)\n", task.UUID, task.Description, task.NotificationDate)
					}
					lastNotifiedDate[task.UUID] = task.NotificationDate
				}
			}
			mu.Unlock()
		}
	}()

	// Notification scheduler (ntfy-based)
	go func() {
		notifier := notify.NewNotifier(cfg.Ntfy, log.Default())
		for {
			time.Sleep(5 * time.Second)
			mu.RLock()
			now := time.Now()
			for _, task := range tasks {
				// Skip if already acknowledged
				if web.IsAcknowledged(task.UUID) {
					continue
				}
				ndates := []string{task.NotificationDate}
				if v, ok := getUDA(task, "taskherald.notification_date"); ok && v != "" {
					ndates = append(ndates, v)
				}
				var notifyAt time.Time
				var notifyDateStr string
				for _, nd := range ndates {
					if nd == "" {
						continue
					}
					t, err := util.ParseNotificationDate(nd)
					if err == nil && (notifyAt.IsZero() || t.Before(notifyAt)) {
						notifyAt = t
						notifyDateStr = nd
					}
				}
				if notifyAt.IsZero() {
					continue
				}
				// Only process notifications that are due right now (within the last 5 minutes)
				nowLocal := now.In(time.Local)
				notifyLocal := notifyAt.In(time.Local)

				// Skip if notification time is in the future
				if notifyLocal.After(nowLocal) {
					continue
				}

				// Only process if notification is due within the last 5 minutes (to catch notifications that were missed)
				fiveMinutesAgo := nowLocal.Add(-5 * time.Minute)
				if notifyLocal.Before(fiveMinutesAgo) {
					continue
				}
				// Use UUID|notification_date as the key
				notifyKey := fmt.Sprintf("%s|%s", task.UUID, notifyDateStr)
				if _, already := notified[notifyKey]; already {
					continue
				}
				// Log the notification time in both UTC and local
				fmt.Printf("[notify] Task %s will be notified at local: %s (UTC: %s)\n", task.UUID, notifyAt.In(time.Local).Format("2006-01-02 15:04:05 MST"), notifyAt.UTC().Format("2006-01-02 15:04:05 UTC"))
				// Prepare message
				msgTmpl := cfg.NotificationMessage
				info := notify.TaskInfo{
					ID:               fmt.Sprintf("%d", task.ID),
					UUID:             task.UUID,
					Description:      task.Description,
					Tags:             task.Tags,
					Project:          task.Project,
					Priority:         task.Priority,
					NotificationDate: &notifyAt,
				}
				msg, err := notify.RenderMessage(info, msgTmpl)
				if err != nil {
					msg = fmt.Sprintf("Task %s: %s", task.ID, task.Description)
				}
				// Prepare dynamic headers (e.g., X-Title, X-Click, X-Actions)
				headers := map[string]string{}
				for k, v := range cfg.Ntfy.Headers {
					headers[k] = v
				}
				// Set X-Title to project
				if task.Project != "" {
					headers["X-Title"] = task.Project
				}
				// Map Taskwarrior priority to ntfy priority for X-Default
				var ntfyPriority string
				switch task.Priority {
				case "H", "h":
					ntfyPriority = "max"
				case "M", "m":
					ntfyPriority = "high"
				case "L", "l":
					ntfyPriority = "default"
				default:
					ntfyPriority = "default"
				}
				headers["X-Default"] = ntfyPriority

				// If actions_enabled, set X-Actions header automatically
				if cfg.Ntfy.ActionsEnabled {
					// Determine scheme, domain, and port
					scheme := "http"
					domain := cfg.Web.Domain
					port := ""
					listen := cfg.Web.Listen
					// Extract port from listen (e.g., 127.0.0.1:8080 or :8080)
					if listen != "" {
						// Try net.SplitHostPort, fallback for :8080
						var p string
						if listen[0] == ':' {
							// e.g., :8080
							p = listen[1:]
						} else {
							// Try to split host:port
							hostPort := listen
							if _, prt, err := net.SplitHostPort(hostPort); err == nil {
								p = prt
							} else {
								// fallback: try last colon
								if idx := len(listen) - 1; idx >= 0 {
									for j := idx; j >= 0; j-- {
										if listen[j] == ':' {
											p = listen[j+1:]
											break
										}
									}
								}
							}
						}
						if p != "" {
							port = p
						}
					}
					if domain == "" {
						domain = "localhost"
					}
					url := scheme + "://" + domain
					if port != "" {
						url += ":" + port
					}
					url += "/delay?uuid=" + task.UUID + "&minutes=30"
					headers["X-Actions"] = `[{"action":"view","label":"Delay 30m","url":"` + url + `"}]`
				}
				// Optionally, interpolate fields in headers here if needed
				// Send notification
				err = notifier.Send(nil, msg, headers)
				nowLocalMsg := time.Now().In(time.Local)
				if err == nil {
					notified[notifyKey] = struct{}{}
					fmt.Printf("[notify] Notification sent for task %s at %s\n", task.UUID, nowLocalMsg.Format("2006-01-02 15:04:05 MST"))
				} else if err != nil {
					fmt.Printf("[notify] Failed to send notification for task %s: %v\n", task.UUID, err)
				}
			}
			mu.RUnlock()
		}
	}()

	// Start web server
	webListen := cfg.Web.Listen
	if webListen == "" {
		webListen = ":8080"
	}
	fmt.Printf("[startup] Starting web UI on %s...\n", webListen)
	server := web.NewServer(func() []taskwarrior.Task {
		mu.RLock()
		defer mu.RUnlock()
		return append([]taskwarrior.Task(nil), tasks...)
	})
	go func() {
		err := server.Serve(webListen)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[error] Web server failed: %v\n", err)
			os.Exit(2)
		}
	}()

	// Block until interrupted (SIGINT/SIGTERM)
	sigCh := make(chan struct{})
	select {
	case <-sigCh:
	}
	return nil
}

// getUDA returns the value of a UDA if present in the task struct (by field name)
func getUDA(task taskwarrior.Task, field string) (string, bool) {
	cfg := config.Get()
	var fieldName string
	switch field {
	case "notification_date":
		fieldName = cfg.UDAMap.NotificationDate
	case "repeat_enable":
		fieldName = cfg.UDAMap.RepeatEnable
	case "repeat_delay":
		fieldName = cfg.UDAMap.RepeatDelay
	default:
		fieldName = field
	}
	// Direct struct field for notification_date
	if fieldName == "notification_date" {
		return task.NotificationDate, task.NotificationDate != ""
	}
	// For other UDAs, try to get from map (future extensibility)
	v, ok := getTaskField(&task, fieldName)
	return v, ok && v != ""
}

// getTaskField uses reflection to get a field by name from Task struct or its map (if present)
func getTaskField(task *taskwarrior.Task, field string) (string, bool) {
	// If you add fields to Task struct, add them here
	// For now, try to use struct tags if present, else fallback to map (if you add one)
	// This is a placeholder for future extensibility
	// If you use map[string]interface{} for UDAs, handle here
	return "", false
}

// parseNotificationDate parses a date string in common Taskwarrior formats
