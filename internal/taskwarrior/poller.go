package taskwarrior

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"unicode"
	"task-herald/internal/config"
	"time"
)

type Task struct {
	ID               int      `json:"id"`
	UUID             string   `json:"uuid"`
	Description      string   `json:"description"`
	NotificationDate string   `json:"notification_date"`
	Tags             []string `json:"tags"`
	Priority         string   `json:"priority"`
	Project          string   `json:"project"`
	Status           string   `json:"status"`
}

// ParseNotificationDate parses the NotificationDate string into a time.Time object.
func (t *Task) ParseNotificationDate() (time.Time, error) {
	// Try Taskwarrior's default date format: "2006-01-02 15:04:05"
	if t.NotificationDate == "" {
		return time.Time{}, fmt.Errorf("NotificationDate is empty")
	}
	layouts := []string{
		"2006-01-02 15:04:05",
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04",
		"2006-01-02T15:04",
	}
	var lastErr error
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, t.NotificationDate)
		if err == nil {
			return parsed, nil
		}
		lastErr = err
	}
	return time.Time{}, fmt.Errorf("could not parse NotificationDate '%s': %v", t.NotificationDate, lastErr)
}

func ExportIncompleteTasks() ([]Task, error) {
	// Get pending tasks
	pendingCmd := exec.Command("task", "status:pending", "export", "rc.json.array=on")
	var pendingOut bytes.Buffer
	pendingCmd.Stdout = &pendingOut
	pendingCmd.Stderr = &pendingOut
	err := pendingCmd.Run()
	pendingRaw := pendingOut.String()
	// DEBUG: Log raw response from task export (pending)
	config.Log(config.DEBUG, "DEBUG: Raw task export (pending): %s", pendingRaw)
	if err != nil {
		return nil, err
	}
	start := -1
	end := -1
	for i, c := range pendingRaw {
		if c == '[' && start == -1 {
			start = i
		}
		if c == ']' {
			end = i
		}
	}
	if start == -1 || end == -1 || end <= start {
		config.Log(config.DEBUG, "DEBUG: Could not find JSON array in pending output")
		return nil, fmt.Errorf("could not find JSON array in pending output")
	}
	pendingPart := pendingRaw[start : end+1]
	var pendingTasks []Task
	if err := json.Unmarshal([]byte(pendingPart), &pendingTasks); err != nil {
		config.Log(config.DEBUG, "DEBUG: JSON unmarshal error (pending): %v", err)
		return nil, err
	}

	// Get waiting tasks
	waitingCmd := exec.Command("task", "status:waiting", "export", "rc.json.array=on")
	var waitingOut bytes.Buffer
	waitingCmd.Stdout = &waitingOut
	waitingCmd.Stderr = &waitingOut
	err = waitingCmd.Run()
	waitingRaw := waitingOut.String()
	// DEBUG: Log raw response from task export (waiting)
	config.Log(config.DEBUG, "DEBUG: Raw task export (waiting): %s", waitingRaw)
	if err != nil {
		return nil, err
	}
	start = -1
	end = -1
	for i, c := range waitingRaw {
		if c == '[' && start == -1 {
			start = i
		}
		if c == ']' {
			end = i
		}
	}
	if start == -1 || end == -1 || end <= start {
		config.Log(config.DEBUG, "DEBUG: Could not find JSON array in waiting output")
		return nil, fmt.Errorf("could not find JSON array in waiting output")
	}
	waitingPart := waitingRaw[start : end+1]
	var waitingTasks []Task
	if err := json.Unmarshal([]byte(waitingPart), &waitingTasks); err != nil {
		config.Log(config.DEBUG, "DEBUG: JSON unmarshal error (waiting): %v", err)
		return nil, err
	}

	// Combine both
	allTasks := append(pendingTasks, waitingTasks...)
	config.Log(config.DEBUG, "DEBUG: Parsed %d pending tasks, %d waiting tasks, %d total", len(pendingTasks), len(waitingTasks), len(allTasks))

	// After combining all tasks, process +tag in description
	tagPattern := regexp.MustCompile(`\B\+([a-zA-Z0-9_\-]+)`)
	for _, task := range allTasks {
	       config.Log(config.DEBUG, "Processing task: ID=%d, UUID=%s, Desc=\"%s\"", task.ID, task.UUID, task.Description)
	       matches := tagPattern.FindAllStringSubmatch(task.Description, -1)
	       if len(matches) == 0 {
		       continue
	       }
	       tagsToAdd := make(map[string]struct{})
	       for _, m := range matches {
		       tagsToAdd[m[1]] = struct{}{}
	       }
	       config.Log(config.INFO, "Found +tag(s) in task %s: %v", task.UUID, matches)
	       // Remove +tag from description
	       newDesc := tagPattern.ReplaceAllString(task.Description, "")
	       newDesc = strings.TrimSpace(newDesc)
	       // Add tags and update description in Taskwarrior
	       for tag := range tagsToAdd {
		       normTag := dashToCamel(tag)
		       config.Log(config.INFO, "Adding tag '+%s' (normalized: '+%s') to task %s", tag, normTag, task.UUID)
		       ModifyTask(task.UUID, "+"+normTag)
	       }
	       if newDesc != task.Description {
		       config.Log(config.VERBOSE, "Updating description for task %s: \"%s\" -> \"%s\"", task.UUID, task.Description, newDesc)
		       ModifyTask(task.UUID, "description:"+newDesc)
	       }
	}
	return allTasks, nil
}

// dashToCamel converts dash-separated strings to camelCase (e.g., foo-bar-baz -> fooBarBaz)
func dashToCamel(s string) string {
       var result strings.Builder
       upper := false
       for i, r := range s {
	       if r == '-' {
		       upper = true
		       continue
	       }
	       if upper {
		       result.WriteRune(unicode.ToUpper(r))
		       upper = false
	       } else if i == 0 {
		       result.WriteRune(unicode.ToLower(r))
	       } else {
		       result.WriteRune(r)
	       }
       }
	return result.String()
}

// Poller polls Taskwarrior at the given interval and sends tasks to the channel.
func Poller(interval time.Duration, out chan<- []Task, stop <-chan struct{}) {
	// Fetch tasks immediately on startup
	tasks, err := ExportIncompleteTasks()
	if err == nil {
		out <- tasks
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			tasks, err := ExportIncompleteTasks()
			if err == nil {
				out <- tasks
			}
		case <-stop:
			return
		}
	}
}
