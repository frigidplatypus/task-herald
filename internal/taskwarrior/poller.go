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
	Due              string   `json:"due"`
	NotificationDate string   `json:"notification_date"`
	Modified         string   `json:"modified"`
	Start            string   `json:"start"`
	End              string   `json:"end"`
	Wait             string   `json:"wait"`
	Entry            string   `json:"entry"`
	Tags             []string `json:"tags"`
	Priority         string   `json:"priority"`
	Project          string   `json:"project"`
	Status           string   `json:"status"`
	Urgency          float64  `json:"urgency"`
}

func ExportIncompleteTasks() ([]Task, error) {
	// Only status:pending tasks (active/incomplete)
		// Export all fields including computed urgency
		cmd := exec.Command("task", "status:pending", "export", "rc.json.array=on", "rc.json.allfields=yes")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	raw := out.String()
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
		return nil, fmt.Errorf("could not find JSON array in output")
	}
	jsonPart := raw[start : end+1]
	var tasks []Task
	if err := json.Unmarshal([]byte(jsonPart), &tasks); err != nil {
		return nil, err
	}
	// Populate urgency for each task by re-exporting per UUID
	for i := range tasks {
		// Export single task to get computed urgency
		uCmd := exec.Command("task", tasks[i].UUID, "export", "rc.json.array=on")
		var uBuf bytes.Buffer
		uCmd.Stdout = &uBuf
		uCmd.Stderr = &uBuf
		if err := uCmd.Run(); err == nil {
			rawU := uBuf.String()
			// extract JSON array
			sIdx, eIdx := -1, -1
			for j, ch := range rawU {
				if ch == '[' && sIdx < 0 {
					sIdx = j
				}
				if ch == ']' {
					eIdx = j
				}
			}
			if sIdx >= 0 && eIdx > sIdx {
				var single []Task
				if err := json.Unmarshal([]byte(rawU[sIdx:eIdx+1]), &single); err == nil && len(single) > 0 {
					tasks[i].Urgency = single[0].Urgency
				}
			}
		}
	}
	// Debug: log urgency values fetched
	for _, t := range tasks {
		fmt.Printf("[DEBUG] Task %s urgency=%f\n", t.UUID, t.Urgency)
	}
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
