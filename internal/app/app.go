package app

import (
	"fmt"
	"log"
	"sync"
	"task-herald/internal/config"
	"task-herald/internal/notify"
	"task-herald/internal/taskwarrior"
	"task-herald/internal/util"
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
		notified = make(map[string]struct{}) // Key: UUID|notification_date
		lastNotifiedDate = make(map[string]string) // Key: UUID, Value: last seen notification_date
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

       // Notification scheduler
       go func() {
	       notifier := notify.NewNotifier(cfg.ShoutrrrURL, log.Default())
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
			       msg = fmt.Sprintf("Task %s: %s", task.ID, task.Description)
		       }
		       // Send notification
		       err = notifier.Send(nil, msg)
		       nowLocalMsg := time.Now().In(time.Local)
	       if err == nil {
		       notified[notifyKey] = struct{}{}
		       fmt.Printf("[notify] Notification sent for task %s at %s via %s\n", task.UUID, nowLocalMsg.Format("2006-01-02 15:04:05 MST"), cfg.ShoutrrrURL)
	       } else if err != nil {
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
