package app

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"
	"text/template"
	"net/url"
	"task-herald/internal/config"
	"task-herald/internal/notify"
	"task-herald/internal/taskwarrior"
	"task-herald/internal/util"
	"time"
)

// Overridable hooks for testing
var (
	syncOnceFunc = func() { taskwarrior.SyncOnce() }
	pollerFunc   = func(interval time.Duration, out chan<- []taskwarrior.Task, stop <-chan struct{}) {
		taskwarrior.Poller(interval, out, stop)
	}
	syncTaskwarriorFunc = func(stop <-chan struct{}) { taskwarrior.SyncTaskwarrior(stop) }
	runSigCh            = func() <-chan struct{} { return make(chan struct{}) }
)

// Additional overridable hooks for testing
var (
	loadConfigFunc = config.LoadConfig
	newNotifierFunc = func(cfg config.NtfyConfig, logger func(format string, v ...interface{})) typeNotifier {
		return notify.NewNotifier(cfg, logger)
	}
	notifySleepDuration = 5 * time.Second
)

// typeNotifier is an interface used so tests can inject a fake notifier
type typeNotifier interface {
	Send(ctx context.Context, message string, headers map[string]string) error
}

func Run(configOverride string) error {
	config.Log(config.INFO, "Taskwarrior Notifications service starting...")

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

	cfg, err := loadConfigFunc(cfgPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	config.Set(cfg)

	// DEBUG: Log parsed config struct
	config.Log(config.DEBUG, "Loaded config: %+v", *cfg)

	// DEBUG: Log relevant environment variables
	config.Log(config.DEBUG, "TASK_HERALD_CONFIG env: %s", os.Getenv("TASK_HERALD_CONFIG"))

	// INFO: Log config.yaml location
	config.Log(config.INFO, "Loaded config.yaml from: %s", cfgPath)

	// INFO: Log ntfy.sh server and endpoint
	config.Log(config.INFO, "ntfy.sh server: %s, topic: %s", cfg.Ntfy.URL, cfg.Ntfy.GetTopic())

	// INFO: Log Taskwarrior config/data location (hardcoded for now, could be improved)
	twConfigLoc := "/home/justin/.local/share/task"
	config.Log(config.INFO, "Taskwarrior data/config location: %s", twConfigLoc)

	// Shared state for polled tasks
	var (
		mu               sync.RWMutex
		tasks            []taskwarrior.Task
		notified         = make(map[string]struct{}) // Key: UUID|notification_date
		lastNotifiedDate = make(map[string]string)   // Key: UUID, Value: last seen notification_date
	)

	// Set log level from config
	config.SetLogLevelFromConfig(cfg)

	// Set up polling and syncing
	taskCh := make(chan []taskwarrior.Task)
	stopCh := make(chan struct{})
	// Run 'task sync' immediately at startup
	// Use overridable functions to allow tests to mock behavior
	syncOnceFunc()
	go pollerFunc(cfg.PollInterval, taskCh, stopCh)
	go syncTaskwarriorFunc(stopCh)

	// Update tasks on poll
	go func() {
		for t := range taskCh {
			mu.Lock()
			tasks = t

			// INFO: Log total number of available tasks
			totalTasks := len(t)
			config.Log(config.INFO, "Total available tasks from taskwarrior: %d", totalTasks)

			// INFO: Log all tasks with a future notification_date
			futureTasks := 0
			for _, task := range t {
				if task.NotificationDate == "" {
					continue
				}
				prev := lastNotifiedDate[task.UUID]
				if prev != task.NotificationDate {
					lastNotifiedDate[task.UUID] = task.NotificationDate
				}
				notifyAt, err := util.ParseNotificationDate(task.NotificationDate)
				if err == nil && notifyAt.After(time.Now()) {
					config.Log(config.INFO, "Task with future notification_date: UUID=%s, ID=%d, Desc=\"%s\", Date=%s", task.UUID, task.ID, task.Description, notifyAt.Format("2006-01-02 15:04:05 MST"))
					futureTasks++
				}
			}
			config.Log(config.INFO, "Total tasks with future notification_date: %d", futureTasks)

			// VERBOSE: Log all tasks returned by task export
			if config.ParseLogLevel(cfg.LogLevel) >= config.VERBOSE {
				for _, task := range t {
					config.Log(config.VERBOSE, "VERBOSE: Task: %+v", task)
				}
			}

			// DEBUG: Log state of internal maps
			config.Log(config.DEBUG, "DEBUG: notified map: %+v", notified)
			config.Log(config.DEBUG, "DEBUG: lastNotifiedDate map: %+v", lastNotifiedDate)

			mu.Unlock()
		}
	}()

	// Notification scheduler (ntfy-based)
	go func() {
		// Use a logger function that wraps config.Log at INFO level
		loggerFunc := func(format string, v ...interface{}) {
			config.Log(config.INFO, format, v...)
		}
		notifier := newNotifierFunc(cfg.Ntfy, loggerFunc)
		for {
			time.Sleep(notifySleepDuration)
			mu.RLock()
			now := time.Now()
			for _, task := range tasks {
				// Skip if already acknowledged (web interface removed)
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
				config.Log(config.INFO, "[notify] Task %s will be notified at local: %s (UTC: %s)", task.UUID, notifyAt.In(time.Local).Format("2006-01-02 15:04:05 MST"), notifyAt.UTC().Format("2006-01-02 15:04:05 UTC"))
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
					msg = fmt.Sprintf("Task %d: %s", task.ID, task.Description)
				}
				// Prepare dynamic headers (e.g., X-Title, X-Click, X-Actions)
				headers := map[string]string{}
				// Render configured headers as templates against TaskInfo (allow urlquery func)
				funcMap := template.FuncMap{"urlquery": url.QueryEscape}
				for k, v := range cfg.Ntfy.Headers {
					// try to render template; if fails, fall back to raw value
					t, err := template.New(k).Funcs(funcMap).Parse(v)
					if err != nil {
						headers[k] = v
						continue
					}
					var buf strings.Builder
					if err := t.Execute(&buf, info); err != nil {
						headers[k] = v
						continue
					}
					headers[k] = buf.String()
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
				// Send notification
				err = notifier.Send(context.Background(), msg, headers)
				nowLocalMsg := time.Now().In(time.Local)
				if err == nil {
					notified[notifyKey] = struct{}{}
					config.Log(config.INFO, "[notify] Notification sent for task %s at %s", task.UUID, nowLocalMsg.Format("2006-01-02 15:04:05 MST"))
				} else if err != nil {
					config.Log(config.ERROR, "[notify] Failed to send notification for task %s: %v", task.UUID, err)
				}
			}
			mu.RUnlock()
		}
	}()

	// Web server removed

	// Block until interrupted (SIGINT/SIGTERM)
	sigCh := runSigCh()
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
	// Use reflection to get a field by name (case-insensitive) from Task struct
	// Supported fields: ID, UUID, Description, NotificationDate, Tags, Priority, Project, Status
	v := reflect.ValueOf(task).Elem()
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if strings.EqualFold(f.Name, field) || strings.EqualFold(f.Tag.Get("json"), field) {
			fv := v.Field(i)
			// Handle string fields
			switch fv.Kind() {
			case reflect.String:
				return fv.String(), fv.String() != ""
			case reflect.Int, reflect.Int64:
				return fmt.Sprintf("%d", fv.Int()), true
			case reflect.Slice:
				// only handle []string
				if fv.Type().Elem().Kind() == reflect.String {
					ss := make([]string, fv.Len())
					for j := 0; j < fv.Len(); j++ {
						ss[j] = fv.Index(j).String()
					}
					return strings.Join(ss, ","), fv.Len() > 0
				}
			}
		}
	}
	return "", false
}

// parseNotificationDate parses a date string in common Taskwarrior formats
