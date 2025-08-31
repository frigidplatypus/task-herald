
package notify

import (
	"testing"
	"time"
	"context"
	"task-herald/internal/config"
)

type dummyLogger struct {
	calls []string
}

func (d *dummyLogger) Log(format string, v ...interface{}) {
	d.calls = append(d.calls, format)
}

func TestRenderMessage_DefaultTemplate(t *testing.T) {
	tmpl := "Task: {{.Description}} ({{.ID}})"
	info := TaskInfo{
		ID:          "42",
		UUID:        "abc-123",
		Description: "Test task",
		Tags:        []string{"foo", "bar"},
		Project:     "demo",
		Priority:    "H",
		NotificationDate: func() *time.Time { tm := time.Now(); return &tm }(),
	}
	msg, err := RenderMessage(info, tmpl)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg == "" || msg == tmpl {
		t.Errorf("expected rendered message, got %q", msg)
	}
}

func TestNewNotifier_AndSend(t *testing.T) {
   logger := &dummyLogger{}
   ntfyCfg := config.NtfyConfig{
	   URL:   "https://ntfy.example.com",
	   Topic: "test-topic",
   }
   n := NewNotifier(ntfyCfg, logger.Log)
   if n == nil {
	   t.Fatal("Notifier should not be nil")
   }
   // This is a dry test: just check that Send returns an error (since no real server)
   err := n.Send(context.Background(), "test message", map[string]string{"X-Test": "1"})
   if err == nil {
	   t.Error("expected error sending to dummy ntfy server, got nil")
   }
}
