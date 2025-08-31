package taskwarrior

import (
	"os/exec"
	"testing"
	"time"
)

func TestExportIncompleteTasks_CommandError(t *testing.T) {
	origExec := execCommand
	defer func() { execCommand = origExec }()

	execCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("false") // always fails
		// Save and restore execCommand
	}

	_, err := ExportIncompleteTasks()
	if err == nil {
		t.Error("expected error when command fails, got nil")
	}
}

func TestPoller_SendsTasks_AndStops(t *testing.T) {
	origExec := execCommand
	defer func() { execCommand = origExec }()

	pendingJSON := `[{"id":1,"uuid":"abc","description":"pending task","notification_date":"2025-08-31 14:30:00","tags":[],"priority":"H","project":"demo","status":"pending"}]`
	waitingJSON := `[{"id":2,"uuid":"def","description":"waiting task","notification_date":"2025-08-31T14:30:00Z","tags":[],"priority":"L","project":"demo","status":"waiting"}]`
	pendingOut := "" + pendingJSON
	waitingOut := "" + waitingJSON

	call := 0
	execCommand = func(name string, args ...string) *exec.Cmd {
		var out string
		if call == 0 {
			out = pendingOut
		} else {
			out = waitingOut
		}
		call++
		return exec.Command("echo", out)
	}

	outCh := make(chan []Task, 4)
	stop := make(chan struct{})

	go Poller(10*time.Millisecond, outCh, stop)

	// Expect an initial immediate send
	select {
	case tasks := <-outCh:
		if len(tasks) != 2 && len(tasks) != 1 { // combined may produce 2 tasks (pending+waiting)
			t.Fatalf("expected at least 1 task, got %d", len(tasks))
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout waiting for initial poll result")
	}

	// Wait for one more ticked send
	select {
	case <-outCh:
		// ok
	case <-time.After(300 * time.Millisecond):
		t.Fatal("timeout waiting for ticked poll result")
	}

	// Stop the poller
	close(stop)

	// Ensure poller stops and no further sends arrive
	select {
	case <-outCh:
		// If we get another value quickly it's okay, but ensure poller stops within a short window
		// try to wait a bit and ensure no continuous sends
		select {
		case <-outCh:
			t.Fatal("received unexpected additional poll after stop")
		case <-time.After(100 * time.Millisecond):
		}
	case <-time.After(200 * time.Millisecond):
		// no sends - ok
	}
}
func TestExportIncompleteTasks_MalformedOutput(t *testing.T) {
	origExec := execCommand
	defer func() { execCommand = origExec }()

	// Output is not a valid JSON array
	execCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("echo", "not a json array")
	}

	_, err := ExportIncompleteTasks()
	if err == nil {
		t.Error("expected error for malformed output, got nil")
	}
}

func TestParseNotificationDate(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool // want success
	}{
		{"empty", "", false},
		{"default format", "2025-08-31 14:30:00", true},
		{"RFC3339", "2025-08-31T14:30:00Z", true},
		{"short format", "2025-08-31 14:30", true},
		{"short T format", "2025-08-31T14:30", true},
		{"invalid", "not-a-date", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			task := Task{NotificationDate: tc.input}
			_, err := task.ParseNotificationDate()
			if (err == nil) != tc.want {
				t.Errorf("ParseNotificationDate(%q) error = %v, want success: %v", tc.input, err, tc.want)
			}
		})
	}
}

func fakeExecCommandHelper(output string) func(string, ...string) *exec.Cmd {
	return func(name string, args ...string) *exec.Cmd {
		return exec.Command("echo", output)
	}
}

func TestExportIncompleteTasks_Success(t *testing.T) {
	// Save and restore execCommand
	origExec := execCommand
	defer func() { execCommand = origExec }()

	pendingJSON := `[{"id":1,"uuid":"abc","description":"pending task","notification_date":"2025-08-31 14:30:00","tags":["foo"],"priority":"H","project":"demo","status":"pending"}]`
	waitingJSON := `[{"id":2,"uuid":"def","description":"waiting task","notification_date":"2025-08-31T14:30:00Z","tags":["bar"],"priority":"L","project":"demo","status":"waiting"}]`
	// Wrap in extra text to simulate real output
	pendingOut := "Some log...\n" + pendingJSON + "\nMore log..."
	waitingOut := "Other log...\n" + waitingJSON + "\nEnd log."

	call := 0
	execCommand = func(name string, args ...string) *exec.Cmd {
		var out string
		if call == 0 {
			out = pendingOut
		} else {
			out = waitingOut
		}
		call++
		return exec.Command("echo", out)
	}

	tasks, err := ExportIncompleteTasks()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tasks) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(tasks))
	}
	if tasks[0].UUID != "abc" || tasks[1].UUID != "def" {
		t.Errorf("unexpected tasks: %+v", tasks)
	}
}
