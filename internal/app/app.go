package app

import (
	"fmt"
	"log"
	"os"
	"sync"
	"task-herald/internal/config"
	"task-herald/internal/notify"
	"task-herald/internal/taskwarrior"
	"task-herald/internal/util"
	"task-herald/internal/web"
	"context"
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

       // Start web server in a goroutine (if enabled)
       go func() {
	       // Determine address to listen on
	       addr := ""
	       if cfg.Web.Host != "" && cfg.Web.Port != 0 {
		       addr = fmt.Sprintf("%s:%d", cfg.Web.Host, cfg.Web.Port)
	       } else if cfg.Web.Listen != "" {
		       addr = cfg.Web.Listen
	       } else {
		       addr = "127.0.0.1:8080" // fallback default
	       }
	       // Start the web server
	       // Import the web package
	       // NOTE: This import must be at the top of the file:
	       // "task-herald/internal/web"
	       // But for patching, we can add it if not present
	       // Start the server
	       // Use tasks from the closure
	       // If you get an import error, add the import at the top
	       server := web.NewServer(func() []taskwarrior.Task {
		       mu.RLock()
		       defer mu.RUnlock()
		       return tasks
	       })
	       if err := server.Serve(addr); err != nil {
		       fmt.Printf("[web] Server error: %v\n", err)
	       }
       }()

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

       // Notification scheduler
       go func() {
	       shoutrrrURL, err := cfg.GetShoutrrrURL()
	       if err != nil {
		       fmt.Printf("Error getting shoutrrr URL: %v\n", err)
		       return
	       }
	       notifier := notify.NewNotifier(shoutrrrURL, log.Default())
	       for {
		       time.Sleep(5 * time.Second)
		       mu.RLock()
		       now := time.Now()
		       for _, task := range tasks {
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
				       msg = fmt.Sprintf("Task %d: %s", task.ID, task.Description)
			       }
			       // Send notification
			       err = notifier.Send(context.TODO(), msg)
			       nowLocalMsg := time.Now().In(time.Local)
			       if err == nil {
				       notified[notifyKey] = struct{}{}
				       fmt.Printf("[notify] Notification sent for task %s at %s\n", task.UUID, nowLocalMsg.Format("2006-01-02 15:04:05 MST"))
			       } else {
				       fmt.Printf("[notify] Failed to send notification for task %s: %v\n", task.UUID, err)
			       }
		       }
		       mu.RUnlock()
	       }
       }()

	// Block forever (until interrupted)
	select {}
}

// getUDA returns the value of a UDA if present in the task struct (by field name)

// getUDA returns the value of a UDA if present in the task struct (by field name)
func getUDA(task taskwarrior.Task, field string) (string, bool) {
	// Only supports notification_date and taskherald.notification_date for now
	switch field {
	case "notification_date":
		return task.NotificationDate, task.NotificationDate != ""
	case "taskherald.notification_date":
		// If you add this to Task struct, parse here
		return "", false // Not yet implemented in struct
	}

	return "", false
}

// parseNotificationDate parses a date string in common Taskwarrior formats
