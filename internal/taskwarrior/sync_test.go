package taskwarrior

import (
	"os/exec"
	"task-herald/internal/config"
	"testing"
	"time"
)

func TestSyncOnce_Success(t *testing.T) {
	origExec := execCommand
	defer func() { execCommand = origExec }()

	execCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("echo", "Sync completed")
	}

	// Should not panic; we inspect logs manually if needed
	SyncOnce()
}

func TestSyncOnce_Failure(t *testing.T) {
	origExec := execCommand
	defer func() { execCommand = origExec }()

	execCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("false")
	}

	SyncOnce()
}

func TestSyncOnce_Logs(t *testing.T) {
	origExec := execCommand
	defer func() { execCommand = origExec }()
	origLog := logFunc
	defer func() { logFunc = origLog }()

	var entries []string
	logFunc = func(level config.LogLevel, format string, a ...interface{}) {
		entries = append(entries, format)
	}

	execCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("echo", "Sync completed")
	}

	SyncOnce()
	if len(entries) == 0 {
		t.Error("expected log entries from SyncOnce, got none")
	}
}

func TestSyncTaskwarrior_Ticker(t *testing.T) {
	origExec := execCommand
	defer func() { execCommand = origExec }()
	origLog := logFunc
	defer func() { logFunc = origLog }()

	// quick interval
	cfg := &config.Config{SyncInterval: 10 * time.Millisecond}
	config.Set(cfg)

	call := 0
	execCommand = func(name string, args ...string) *exec.Cmd {
		call++
		return exec.Command("echo", "Sync completed")
	}

	// capture logs
	var entries []string
	logFunc = func(level config.LogLevel, format string, a ...interface{}) {
		entries = append(entries, format)
	}

	stop := make(chan struct{})
	go SyncTaskwarrior(stop)

	time.Sleep(100 * time.Millisecond)
	close(stop)

	if call == 0 {
		t.Fatal("expected SyncOnce to be called at least once via ticker")
	}
	if len(entries) == 0 {
		t.Error("expected log entries from SyncTaskwarrior, got none")
	}
}
