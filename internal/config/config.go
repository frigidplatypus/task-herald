package config

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	PollInterval        time.Duration `yaml:"poll_interval"`
	SyncInterval        time.Duration `yaml:"sync_interval"`
	NotificationService string        `yaml:"notification_service"`
	Ntfy                NtfyConfig    `yaml:"ntfy"`
	Web                 WebConfig     `yaml:"web"`
	LogLevel            string        `yaml:"log_level"`
	ShoutrrrURL         string        `yaml:"shoutrrr_url"`
	ShoutrrrURLFile     string        `yaml:"shoutrrr_url_file"`
	NotificationMessage string        `yaml:"notification_message"`
}

type NtfyConfig struct {
	URL   string `yaml:"url"`
	Topic string `yaml:"topic"`
	Token string `yaml:"token"`
}

type WebConfig struct {
	Listen string `yaml:"listen"`
	Host   string `yaml:"host"`
	Port   int    `yaml:"port"`
	Auth   bool   `yaml:"auth"`
}

var (
	currentConfig *Config
	mu            sync.RWMutex
)

func LoadConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var cfg Config
	dec := yaml.NewDecoder(f)
	if err := dec.Decode(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// GetShoutrrrURL returns the shoutrrr URL, reading from file if shoutrrr_url_file is specified
func (c *Config) GetShoutrrrURL() (string, error) {
	if c.ShoutrrrURLFile != "" {
		// Read URL from file
		data, err := os.ReadFile(c.ShoutrrrURLFile)
		if err != nil {
			return "", fmt.Errorf("failed to read shoutrrr_url_file %s: %w", c.ShoutrrrURLFile, err)
		}
		url := strings.TrimSpace(string(data))
		if url == "" {
			return "", fmt.Errorf("shoutrrr_url_file %s is empty", c.ShoutrrrURLFile)
		}
		return url, nil
	}

	if c.ShoutrrrURL == "" {
		return "", fmt.Errorf("neither shoutrrr_url nor shoutrrr_url_file is configured")
	}

	return c.ShoutrrrURL, nil
}

func Get() *Config {
	mu.RLock()
	defer mu.RUnlock()
	return currentConfig
}

func Set(cfg *Config) {
	mu.Lock()
	defer mu.Unlock()
	currentConfig = cfg
}
