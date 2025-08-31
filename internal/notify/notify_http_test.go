package notify

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"task-herald/internal/config"
)

func TestNotifier_Send_HTTPHeadersAndEscaping(t *testing.T) {
	// Start a test HTTP server to capture request and headers
	var gotHeaders http.Header
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeaders = r.Header
		defer r.Body.Close()
		b := make([]byte, r.ContentLength)
		r.Body.Read(b)
		gotBody = b
		w.WriteHeader(200)
	}))
	defer srv.Close()

	cfg := config.NtfyConfig{URL: srv.URL, Topic: "t", Headers: map[string]string{"X-Cfg": "cfgval", "X-Default": "cfg"}}
	n := NewNotifier(cfg, func(format string, v ...interface{}) {})

	// Send message with special characters and override header
	ctx := context.Background()
	msg := "line1\nline2 & <>&{{}}"
	headers := map[string]string{"X-Click": "https://example.com/task/123", "X-Default": "override"}
	if err := n.Send(ctx, msg, headers); err != nil {
		t.Fatalf("Notifier.Send error: %v", err)
	}

	// Verify configured header preserved
	if gotHeaders.Get("X-Cfg") != "cfgval" {
		t.Fatalf("expected X-Cfg header cfgval, got %q", gotHeaders.Get("X-Cfg"))
	}
	// Verify X-Default overridden by send headers
	if gotHeaders.Get("X-Default") != "override" {
		t.Fatalf("expected X-Default override, got %q", gotHeaders.Get("X-Default"))
	}
	// Verify X-Click present
	if gotHeaders.Get("X-Click") != "https://example.com/task/123" {
		t.Fatalf("expected X-Click header, got %q", gotHeaders.Get("X-Click"))
	}
	// Verify body preserved (contains the special sequence)
	if string(gotBody) != msg {
		t.Fatalf("unexpected body: %q", string(gotBody))
	}

	// Also test that non-200 status codes are treated as errors
	badSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer badSrv.Close()
	badCfg := config.NtfyConfig{URL: badSrv.URL, Topic: "t", Headers: map[string]string{}}
	badN := NewNotifier(badCfg, func(format string, v ...interface{}) {})
	err := badN.Send(context.Background(), "x", nil)
	if err == nil {
		t.Fatalf("expected error for non-200 response, got nil")
	}
}
