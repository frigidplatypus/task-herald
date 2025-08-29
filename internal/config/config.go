package config

import (
	"os"
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
	NotificationMessage string        `yaml:"notification_message"`
}

type NtfyConfig struct {
	URL   string `yaml:"url"`
	Topic string `yaml:"topic"`
	Token string `yaml:"token"`
}

type WebConfig struct {
	Listen string `yaml:"listen"`
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
