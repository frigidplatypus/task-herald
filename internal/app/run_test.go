package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"task-herald/internal/config"
	"task-herald/internal/notify"
	"task-herald/internal/taskwarrior"
)

// fakeNotifier tracks Send calls
type fakeNotifier struct {
	mu         sync.Mutex
	calls      int
	lastMsg    string
	lastHeader map[string]string
}

func (f *fakeNotifier) Send(ctx context.Context, message string, headers map[string]string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	f.lastMsg = message
	// copy headers map
	f.lastHeader = make(map[string]string)
	for k, v := range headers {
		f.lastHeader[k] = v
	}
	return nil
}

func TestRun_Integration_ShortLived(t *testing.T) {
	// Override loadConfigFunc to return a minimal config
	origLoad := loadConfigFunc
	defer func() { loadConfigFunc = origLoad }()
	loadConfigFunc = func(path string) (*config.Config, error) {
		return &config.Config{
			PollInterval:        10 * time.Millisecond,
			SyncInterval:        0,
			Ntfy:                config.NtfyConfig{URL: "https://ntfy.example.com", Topic: "test-topic", Headers: map[string]string{}},
			LogLevel:            "debug",
			NotificationMessage: "{{.Description}}",
			UDAMap:              config.UDAMap{NotificationDate: "notification_date"},
		}, nil
	}

	// Override pollerFunc to send a single task then close
	origPoller := pollerFunc
	defer func() { pollerFunc = origPoller }()
	pollerFunc = func(interval time.Duration, out chan<- []taskwarrior.Task, stop <-chan struct{}) {
		out <- []taskwarrior.Task{{ID: 1, UUID: "u1", Description: "t1", NotificationDate: time.Now().Add(-1 * time.Minute).Format(time.RFC3339), Tags: []string{}, Priority: "L", Project: "p1", Status: "pending"}}
		close(out)
	}

	// Override sync functions to no-op
	origSyncOnce := syncOnceFunc
	defer func() { syncOnceFunc = origSyncOnce }()
	syncOnceFunc = func() {}
	origSyncTask := syncTaskwarriorFunc
	defer func() { syncTaskwarriorFunc = origSyncTask }()
	syncTaskwarriorFunc = func(stop <-chan struct{}) {}

	// Override notifier
	origNewNotifier := newNotifierFunc
	defer func() { newNotifierFunc = origNewNotifier }()
	fn := &fakeNotifier{}
	newNotifierFunc = func(cfg config.NtfyConfig, logger func(format string, v ...interface{})) typeNotifier {
		return fn
	}

	// Make Run return after a short time via runSigCh
	origRunSigCh := runSigCh
	defer func() { runSigCh = origRunSigCh }()
	runSigCh = func() <-chan struct{} {
		ch := make(chan struct{})
		// close after 100ms
		go func() { time.Sleep(100 * time.Millisecond); close(ch) }()
		return ch
	}

	// shorten notify sleep to make test fast
	origNotifySleep := notifySleepDuration
	defer func() { notifySleepDuration = origNotifySleep }()
	notifySleepDuration = 10 * time.Millisecond

	if err := Run(""); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if fn.calls == 0 {
		t.Fatalf("expected notifier to be called at least once, got 0")
	}

	// Verify message body and headers
	fn.mu.Lock()
	gotMsg := fn.lastMsg
	gotHeaders := fn.lastHeader
	fn.mu.Unlock()

	if gotMsg != "t1" {
		t.Fatalf("unexpected notification message: got %q, want %q", gotMsg, "t1")
	}
	// Check X-Title header is set to project
	if gotHeaders["X-Title"] != "p1" {
		t.Fatalf("unexpected X-Title header: got %q, want %q", gotHeaders["X-Title"], "p1")
	}
	// Check X-Default header for priority L -> default
	if gotHeaders["X-Default"] != "default" {
		t.Fatalf("unexpected X-Default header: got %q, want %q", gotHeaders["X-Default"], "default")
	}
}

func TestRun_NoProjectHeader(t *testing.T) {
	// Similar to the short-lived test but with empty project
	origLoad := loadConfigFunc
	defer func() { loadConfigFunc = origLoad }()
	loadConfigFunc = func(path string) (*config.Config, error) {
		return &config.Config{
			PollInterval:        10 * time.Millisecond,
			SyncInterval:        0,
			Ntfy:                config.NtfyConfig{URL: "https://ntfy.example.com", Topic: "test-topic", Headers: map[string]string{}},
			LogLevel:            "debug",
			NotificationMessage: "{{.Description}}",
			UDAMap:              config.UDAMap{NotificationDate: "notification_date"},
		}, nil
	}

	origPoller := pollerFunc
	defer func() { pollerFunc = origPoller }()
	pollerFunc = func(interval time.Duration, out chan<- []taskwarrior.Task, stop <-chan struct{}) {
		out <- []taskwarrior.Task{{ID: 2, UUID: "u2", Description: "no-project", NotificationDate: time.Now().Add(-1 * time.Minute).Format(time.RFC3339), Tags: []string{"a"}, Priority: "L", Project: "", Status: "pending"}}
		close(out)
	}

	origSyncOnce := syncOnceFunc
	defer func() { syncOnceFunc = origSyncOnce }()
	syncOnceFunc = func() {}
	origSyncTask := syncTaskwarriorFunc
	defer func() { syncTaskwarriorFunc = origSyncTask }()
	syncTaskwarriorFunc = func(stop <-chan struct{}) {}

	origNewNotifier := newNotifierFunc
	defer func() { newNotifierFunc = origNewNotifier }()
	fn := &fakeNotifier{}
	newNotifierFunc = func(cfg config.NtfyConfig, logger func(format string, v ...interface{})) typeNotifier {
		return fn
	}

	origRunSigCh := runSigCh
	defer func() { runSigCh = origRunSigCh }()
	runSigCh = func() <-chan struct{} {
		ch := make(chan struct{})
		go func() { time.Sleep(100 * time.Millisecond); close(ch) }()
		return ch
	}

	origNotifySleep := notifySleepDuration
	defer func() { notifySleepDuration = origNotifySleep }()
	notifySleepDuration = 10 * time.Millisecond

	if err := Run(""); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if fn.calls == 0 {
		t.Fatalf("expected notifier to be called at least once, got 0")
	}
	fn.mu.Lock()
	_, has := fn.lastHeader["X-Title"]
	fn.mu.Unlock()
	if has {
		t.Fatalf("did not expect X-Title header when project is empty, but it was present: %v", fn.lastHeader)
	}
}

func TestRun_PriorityMappings(t *testing.T) {
	cases := []struct{
		name string
		prio string
		want string
	}{
		{"High","H","max"},
		{"Medium","M","high"},
		{"Low","L","default"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			origLoad := loadConfigFunc
			defer func() { loadConfigFunc = origLoad }()
			loadConfigFunc = func(path string) (*config.Config, error) {
				return &config.Config{
					PollInterval:        10 * time.Millisecond,
					SyncInterval:        0,
					Ntfy:                config.NtfyConfig{URL: "https://ntfy.example.com", Topic: "test-topic", Headers: map[string]string{}},
					LogLevel:            "debug",
					NotificationMessage: "{{.Description}}",
					UDAMap:              config.UDAMap{NotificationDate: "notification_date"},
				}, nil
			}

			origPoller := pollerFunc
			defer func() { pollerFunc = origPoller }()
			pollerFunc = func(interval time.Duration, out chan<- []taskwarrior.Task, stop <-chan struct{}) {
				out <- []taskwarrior.Task{{ID: 3, UUID: "u3", Description: "prio-test", NotificationDate: time.Now().Add(-1 * time.Minute).Format(time.RFC3339), Tags: []string{}, Priority: tc.prio, Project: "proj", Status: "pending"}}
				close(out)
			}

			origSyncOnce := syncOnceFunc
			defer func() { syncOnceFunc = origSyncOnce }()
			syncOnceFunc = func() {}
			origSyncTask := syncTaskwarriorFunc
			defer func() { syncTaskwarriorFunc = origSyncTask }()
			syncTaskwarriorFunc = func(stop <-chan struct{}) {}

			origNewNotifier := newNotifierFunc
			defer func() { newNotifierFunc = origNewNotifier }()
			fn := &fakeNotifier{}
			newNotifierFunc = func(cfg config.NtfyConfig, logger func(format string, v ...interface{})) typeNotifier {
				return fn
			}

			origRunSigCh := runSigCh
			defer func() { runSigCh = origRunSigCh }()
			runSigCh = func() <-chan struct{} {
				ch := make(chan struct{})
				go func() { time.Sleep(100 * time.Millisecond); close(ch) }()
				return ch
			}

			origNotifySleep := notifySleepDuration
			defer func() { notifySleepDuration = origNotifySleep }()
			notifySleepDuration = 10 * time.Millisecond

			if err := Run(""); err != nil {
				t.Fatalf("Run returned error: %v", err)
			}

			if fn.calls == 0 {
				t.Fatalf("expected notifier to be called at least once, got 0")
			}
			fn.mu.Lock()
			got := fn.lastHeader["X-Default"]
			fn.mu.Unlock()
			if got != tc.want {
				t.Fatalf("priority %s: unexpected X-Default header: got %q, want %q", tc.prio, got, tc.want)
			}
		})
	}
}

func TestRun_HeaderMerging(t *testing.T) {
	// Test that configured Ntfy.Headers are merged into the final headers
	origLoad := loadConfigFunc
	defer func() { loadConfigFunc = origLoad }()
	loadConfigFunc = func(path string) (*config.Config, error) {
		return &config.Config{
			PollInterval:        10 * time.Millisecond,
			SyncInterval:        0,
			Ntfy:                config.NtfyConfig{URL: "https://ntfy.example.com", Topic: "test-topic", Headers: map[string]string{"X-Custom": "v1", "X-Default": "override"}},
			LogLevel:            "debug",
			NotificationMessage: "{{.Description}}",
			UDAMap:              config.UDAMap{NotificationDate: "notification_date"},
		}, nil
	}

	origPoller := pollerFunc
	defer func() { pollerFunc = origPoller }()
	pollerFunc = func(interval time.Duration, out chan<- []taskwarrior.Task, stop <-chan struct{}) {
		out <- []taskwarrior.Task{{ID: 4, UUID: "u4", Description: "hdr-test", NotificationDate: time.Now().Add(-1 * time.Minute).Format(time.RFC3339), Tags: []string{}, Priority: "L", Project: "proj", Status: "pending"}}
		close(out)
	}

	origSyncOnce := syncOnceFunc
	defer func() { syncOnceFunc = origSyncOnce }()
	syncOnceFunc = func() {}
	origSyncTask := syncTaskwarriorFunc
	defer func() { syncTaskwarriorFunc = origSyncTask }()
	syncTaskwarriorFunc = func(stop <-chan struct{}) {}

	origNewNotifier := newNotifierFunc
	defer func() { newNotifierFunc = origNewNotifier }()
	fn := &fakeNotifier{}
	newNotifierFunc = func(cfg config.NtfyConfig, logger func(format string, v ...interface{})) typeNotifier {
		return fn
	}

	origRunSigCh := runSigCh
	defer func() { runSigCh = origRunSigCh }()
	runSigCh = func() <-chan struct{} {
		ch := make(chan struct{})
		go func() { time.Sleep(100 * time.Millisecond); close(ch) }()
		return ch
	}

	origNotifySleep := notifySleepDuration
	defer func() { notifySleepDuration = origNotifySleep }()
	notifySleepDuration = 10 * time.Millisecond

	if err := Run(""); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if fn.calls == 0 {
		t.Fatalf("expected notifier to be called at least once, got 0")
	}

	fn.mu.Lock()
	gotHeaders := fn.lastHeader
	fn.mu.Unlock()

	// Custom header should be present
	if gotHeaders["X-Custom"] != "v1" {
		t.Fatalf("expected X-Custom header v1, got %q", gotHeaders["X-Custom"])
	}
	// Configured X-Default should be overridden by priority mapping to "default"
	if gotHeaders["X-Default"] != "default" {
		t.Fatalf("expected X-Default to be 'default' after priority mapping, got %q", gotHeaders["X-Default"])
	}
}

func TestRun_XActionsTemplate(t *testing.T) {
	// Create temporary config file with X-Actions header template that needs url-escaping
		// cfgYaml unused (we override loadConfigFunc below)
	// Start test server to receive POST
	var gotHeaders map[string][]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeaders = r.Header
		w.WriteHeader(200)
	}))
	defer srv.Close()

	// Override loadConfigFunc to return our constructed config
	origLoad := loadConfigFunc
	defer func() { loadConfigFunc = origLoad }()
	loadConfigFunc = func(path string) (*config.Config, error) {
		return &config.Config{
			PollInterval:        10 * time.Millisecond,
			SyncInterval:        0,
			Ntfy: config.NtfyConfig{
				URL:    srv.URL,
				Topic:  "t",
				Headers: map[string]string{"X-Actions": `[{"action":"view","label":"View","url":"https://example.com/task/{{.UUID | urlquery}}"}]`},
			},
			LogLevel:            "debug",
			NotificationMessage: "{{.Description}}",
			UDAMap:              config.UDAMap{NotificationDate: "notification_date"},
		}, nil
	}
	// Provide poller that sends a task and closes
	origPoller := pollerFunc
	defer func() { pollerFunc = origPoller }()
	pollerFunc = func(interval time.Duration, out chan<- []taskwarrior.Task, stop <-chan struct{}) {
		out <- []taskwarrior.Task{{ID: 5, UUID: "u/with spaces", Description: "act", NotificationDate: time.Now().Add(-1 * time.Minute).Format(time.RFC3339), Tags: []string{}, Priority: "L", Project: "p", Status: "pending"}}
		close(out)
	}

	origSyncOnce := syncOnceFunc
	defer func() { syncOnceFunc = origSyncOnce }()
	syncOnceFunc = func() {}
	origSyncTask := syncTaskwarriorFunc
	defer func() { syncTaskwarriorFunc = origSyncTask }()
	syncTaskwarriorFunc = func(stop <-chan struct{}) {}

	// Replace notifier with real one pointing at our test server
	origNewNotifier := newNotifierFunc
	defer func() { newNotifierFunc = origNewNotifier }()
	newNotifierFunc = func(cfg config.NtfyConfig, logger func(format string, v ...interface{})) typeNotifier {
		return notify.NewNotifier(cfg, logger)
	}

	origRunSigCh := runSigCh
	defer func() { runSigCh = origRunSigCh }()
	runSigCh = func() <-chan struct{} {
		ch := make(chan struct{})
		go func() { time.Sleep(200 * time.Millisecond); close(ch) }()
		return ch
	}

	origNotifySleep := notifySleepDuration
	defer func() { notifySleepDuration = origNotifySleep }()
	notifySleepDuration = 20 * time.Millisecond

	if err := Run(""); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	// Ensure X-Actions header contains URL-escaped UUID
	if gotHeaders == nil {
		t.Fatalf("no request received by server")
	}
	xa := gotHeaders["X-Actions"]
	if len(xa) == 0 {
		t.Fatalf("no X-Actions header sent")
	}
	if !strings.Contains(xa[0], "u%2Fwith+spaces") {
		t.Fatalf("expected escaped uuid in X-Actions header, got %q", xa[0])
	}
}

func TestRun_StartHTTPServer(t *testing.T) {
	// override loadConfigFunc to minimal config
	origLoad := loadConfigFunc
	defer func() { loadConfigFunc = origLoad }()
	loadConfigFunc = func(path string) (*config.Config, error) {
		return &config.Config{PollInterval: 10 * time.Millisecond, SyncInterval: 0, Ntfy: config.NtfyConfig{URL: "https://ntfy.example.com", Topic: "test-topic", Headers: map[string]string{}}, LogLevel: "debug", NotificationMessage: "{{.Description}}", UDAMap: config.UDAMap{NotificationDate: "notification_date"}}, nil
	}

	// override startHTTPServerFunc to start an httptest.Server and signal when ready
	origStart := startHTTPServerFunc
	defer func() { startHTTPServerFunc = origStart }()
	started := make(chan string, 1)
	startHTTPServerFunc = func(handler http.Handler) (func() error, string, error) {
		srv := httptest.NewServer(handler)
		// signal server URL
		started <- srv.URL
		return func() error { srv.Close(); return nil }, srv.Listener.Addr().String(), nil
	}

	// override poller to no-op
	origPoller := pollerFunc
	defer func() { pollerFunc = origPoller }()
	pollerFunc = func(interval time.Duration, out chan<- []taskwarrior.Task, stop <-chan struct{}) {
		out <- []taskwarrior.Task{}
		close(out)
	}

	// prepare runSigCh that we control
	origRunSigCh := runSigCh
	defer func() { runSigCh = origRunSigCh }()
	sigCh := make(chan struct{})
	runSigCh = func() <-chan struct{} { return sigCh }

	// run in background
	runErr := make(chan error, 1)
	go func() {
		runErr <- Run("")
	}()

	// wait for server to be started
	var serverURL string
	select {
	case serverURL = <-started:
	case <-time.After(2 * time.Second):
		t.Fatalf("server did not start in time")
	}

	// verify health endpoint while Run is running
	resp, err := http.Get(serverURL + "/api/health")
	if err != nil {
		// stop Run
		close(sigCh)
		t.Fatalf("http get health failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		close(sigCh)
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// now signal Run to exit and wait
	close(sigCh)
	if err := <-runErr; err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
}
