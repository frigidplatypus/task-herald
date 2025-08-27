package taskwarrior

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
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

func ExportIncompleteTasks() ([]Task, error) {
	// Only status:pending tasks (active/incomplete)
	cmd := exec.Command("task", "status:pending", "export", "rc.json.array=on")
       var out bytes.Buffer
       cmd.Stdout = &out
       cmd.Stderr = &out
       err := cmd.Run()
       raw := out.String()
       fmt.Printf("[DEBUG] Ran: %v\n", cmd.Args)
       fmt.Printf("[DEBUG] Raw Output:\n%s\n", raw)
       fmt.Printf("[DEBUG] Error: %v\n", err)
       if err != nil {
	       return nil, err
       }
       // Extract only the JSON array (from first '[' to last ']')
       start := -1
       end := -1
       for i, c := range raw {
	       if c == '[' && start == -1 {
		       start = i
	       }
	       if c == ']' {
		       end = i
	       }
       }
       if start == -1 || end == -1 || end <= start {
	       fmt.Printf("[DEBUG] Could not find JSON array in output\n")
	       return nil, fmt.Errorf("could not find JSON array in output")
       }
       jsonPart := raw[start : end+1]
       var tasks []Task
       if err := json.Unmarshal([]byte(jsonPart), &tasks); err != nil {
	       fmt.Printf("[DEBUG] JSON unmarshal error: %v\n", err)
	       return nil, err
       }
       fmt.Printf("[DEBUG] Parsed %d tasks\n", len(tasks))
       return tasks, nil
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
