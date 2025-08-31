package notify

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"task-herald/internal/config"
)

// Notifier sends notifications to ntfy using HTTP POST and headers for features
type Notifier struct {
	cfg    config.NtfyConfig
	logger func(format string, v ...interface{})
}

// logger is a printf-style function (e.g., config.Log or a wrapper)
func NewNotifier(cfg config.NtfyConfig, logger func(format string, v ...interface{})) *Notifier {
	return &Notifier{cfg: cfg, logger: logger}
}

// Send sends a message to ntfy with optional headers for advanced features
func (n *Notifier) Send(ctx context.Context, message string, headers map[string]string) error {
	url := fmt.Sprintf("%s/%s", n.cfg.URL, n.cfg.GetTopic())
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer([]byte(message)))
	if err != nil {
		if n.logger != nil {
			n.logger("[notify] failed to create request: %v", err)
		}
		return err
	}
	req.Header.Set("Content-Type", "text/plain")
	if n.cfg.Token != "" {
		req.Header.Set("Authorization", "Bearer "+n.cfg.Token)
	}
	// Set extra headers from config and call
	for k, v := range n.cfg.Headers {
		req.Header.Set(k, v)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if n.logger != nil {
			n.logger("[notify] failed to send notification: %v", err)
		}
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		if n.logger != nil {
			n.logger("[notify] ntfy server returned status: %s", resp.Status)
		}
		return fmt.Errorf("ntfy server returned status: %s", resp.Status)
	}
	return nil
}
