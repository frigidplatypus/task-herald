
package taskwarrior

import (
       "github.com/yourusername/task-herald/internal/config"
       "regexp"
       "strings"
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
       // ...existing code...
       // Get pending tasks
       pendingCmd := exec.Command("task", "status:pending", "export", "rc.json.array=on")
       var pendingOut bytes.Buffer
       pendingCmd.Stdout = &pendingOut
       pendingCmd.Stderr = &pendingOut
       err := pendingCmd.Run()
       pendingRaw := pendingOut.String()
       fmt.Printf("[DEBUG] Ran: %v\n", pendingCmd.Args)
       fmt.Printf("[DEBUG] Raw Output:\n%s\n", pendingRaw)
       fmt.Printf("[DEBUG] Error: %v\n", err)
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
	       fmt.Printf("[DEBUG] Could not find JSON array in pending output\n")
	       return nil, fmt.Errorf("could not find JSON array in pending output")
       }
       pendingPart := pendingRaw[start : end+1]
       var pendingTasks []Task
       if err := json.Unmarshal([]byte(pendingPart), &pendingTasks); err != nil {
	       fmt.Printf("[DEBUG] JSON unmarshal error (pending): %v\n", err)
	       return nil, err
       }

       // Get waiting tasks
       waitingCmd := exec.Command("task", "status:waiting", "export", "rc.json.array=on")
       var waitingOut bytes.Buffer
       waitingCmd.Stdout = &waitingOut
       waitingCmd.Stderr = &waitingOut
       err = waitingCmd.Run()
       waitingRaw := waitingOut.String()
       fmt.Printf("[DEBUG] Ran: %v\n", waitingCmd.Args)
       fmt.Printf("[DEBUG] Raw Output:\n%s\n", waitingRaw)
       fmt.Printf("[DEBUG] Error: %v\n", err)
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
	       fmt.Printf("[DEBUG] Could not find JSON array in waiting output\n")
	       return nil, fmt.Errorf("could not find JSON array in waiting output")
       }
       waitingPart := waitingRaw[start : end+1]
       var waitingTasks []Task
       if err := json.Unmarshal([]byte(waitingPart), &waitingTasks); err != nil {
	       fmt.Printf("[DEBUG] JSON unmarshal error (waiting): %v\n", err)
	       return nil, err
       }

       // Combine both
       allTasks := append(pendingTasks, waitingTasks...)
       fmt.Printf("[DEBUG] Parsed %d pending, %d waiting, %d total tasks\n", len(pendingTasks), len(waitingTasks), len(allTasks))

       // Log all parsed notification dates
       for _, task := range allTasks {
              if task.NotificationDate != "" {
                     config.Log(config.INFO, "Task ID=%d, UUID=%s, NotificationDate=%s", task.ID, task.UUID, task.NotificationDate)
              }
       }

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
                     config.Log(config.INFO, "Adding tag '+%s' to task %s", tag, task.UUID)
                     ModifyTask(task.UUID, "+"+tag)
              }
              if newDesc != task.Description {
                     config.Log(config.VERBOSE, "Updating description for task %s: \"%s\" -> \"%s\"", task.UUID, task.Description, newDesc)
                     ModifyTask(task.UUID, "description:"+newDesc)
              }
       }
       return allTasks, nil
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
