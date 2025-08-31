package taskwarrior

import (
	"strings"
	"task-herald/internal/config"
	"time"
)

// logFunc allows tests to capture log output. Defaults to config.Log.
var logFunc = config.Log

// SyncTaskwarrior runs 'task sync' every 5 minutes in a background goroutine.
func SyncTaskwarrior(stop <-chan struct{}) {
	interval := config.Get().SyncInterval
	if interval == 0 {
		interval = 5 * time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			SyncOnce()
		case <-stop:
			return
		}
	}
}

// SyncOnce runs 'task sync' one time immediately.
func SyncOnce() {
	logFunc(config.INFO, "Running task sync...")
	cmd := execCommand("task", "sync")
	output, err := cmd.CombinedOutput()
	if err != nil {
		logFunc(config.ERROR, "task sync failed: %v\nOutput: %s", err, string(output))
		return
	}
	if !strings.Contains(string(output), "Sync completed") && !strings.Contains(string(output), "synchronized") && !strings.Contains(string(output), "Syncing with sync server") {
		logFunc(config.WARN, "task sync: command ran but did not confirm sync: %s", string(output))
		return
	}
	logFunc(config.INFO, "task sync succeeded: %s", string(output))
}
