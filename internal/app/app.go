package app

import (
	"fmt"
	"log"
	"sync"
	"task-herald/internal/config"
	"task-herald/internal/notify"
	"task-herald/internal/taskwarrior"
	"time"
)

func Run() error {
	fmt.Println("Taskwarrior Notifications service starting...")

	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	config.Set(cfg)

	// Shared state for polled tasks
	var (
		mu       sync.RWMutex
		tasks    []taskwarrior.Task
		notified = make(map[string]struct{}) // UUIDs already notified
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
			mu.Unlock()
			fmt.Printf("Polled %d tasks with notification_date set\n", len(t))
		}
	}()

       // Notification scheduler
       go func() {
	       notifier := notify.NewNotifier(cfg.ShoutrrrURL, log.Default())
	       for {
		       time.Sleep(5 * time.Second)
		       mu.RLock()
		       now := time.Now()
		       for _, task := range tasks {
			       // Support both notification_date and taskherald.notification_date
			       ndates := []string{task.NotificationDate}
			       if v, ok := getUDA(task, "taskherald.notification_date"); ok && v != "" {
				       ndates = append(ndates, v)
			       }
			       var notifyAt time.Time
			       for _, nd := range ndates {
				       if nd == "" {
					       continue
				       }
				       t, err := parseNotificationDate(nd)
				       if err == nil && (notifyAt.IsZero() || t.Before(notifyAt)) {
					       notifyAt = t
				       }
			       }
			       if notifyAt.IsZero() || notifyAt.After(now) {
				       continue
			       }
			       if _, already := notified[task.UUID]; already {
				       continue
			       }
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
			       // Send notification
			       err = notifier.Send(nil, msg)
			       if err == nil {
				       notified[task.UUID] = struct{}{}
				       fmt.Printf("[notify] Sent notification for task %s\n", task.UUID)
			       } else {
				       fmt.Printf("[notify] Failed to send notification for task %s: %v\n", task.UUID, err)
			       }
		       }
		       mu.RUnlock()
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
func parseNotificationDate(s string) (time.Time, error) {
	// Try RFC3339, then "2006-01-02T15:04:05", then "2006-01-02 15:04"
	layouts := []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02 15:04"}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("could not parse notification date: %s", s)
}
