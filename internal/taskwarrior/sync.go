package taskwarrior

import (
	"os/exec"
	"strings"
	"task-herald/internal/config"
	"time"
)

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
       config.Log(config.INFO, "Running task sync...")
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
