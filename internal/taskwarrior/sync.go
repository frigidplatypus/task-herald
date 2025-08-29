package taskwarrior

import (
	"os/exec"
	"strings"
	"time"
	"github.com/yourusername/task-herald/internal/config"
)

// SyncTaskwarrior runs 'task sync' every 5 minutes in a background goroutine.
func SyncTaskwarrior(stop <-chan struct{}) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			syncOnce()
		case <-stop:
			return
		}
	}
}

func syncOnce() {
	cmd := exec.Command("task", "sync")
	output, err := cmd.CombinedOutput()
	if err != nil {
		config.Log(config.ERROR, "task sync failed: %v\nOutput: %s", err, string(output))
		return
	}
	if !strings.Contains(string(output), "Sync completed") && !strings.Contains(string(output), "synchronized") && !strings.Contains(string(output), "Syncing with sync server") {
		config.Log(config.WARN, "task sync: command ran but did not confirm sync: %s", string(output))
		return
	}
	config.Log(config.INFO, "task sync succeeded: %s", string(output))
}
