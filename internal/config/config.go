package config

import (
	"gopkg.in/yaml.v3"
	"os"
	"sync"
	"time"
)

type Config struct {
		PollInterval        time.Duration   `yaml:"poll_interval"`
		Shoutrrr            ShoutrrrConfig  `yaml:"shoutrrr"`
	Web                 WebConfig     `yaml:"web"`
	// Shoutrrr notification endpoints
	Shoutrrr            ShoutrrrConfig `yaml:"shoutrrr"`
}

// ShoutrrrConfig holds Shoutrrr notification URLs
type ShoutrrrConfig struct {
	URLs []string `yaml:"urls"`
}

// NtfyConfig is deprecated; use Shoutrrr URLs instead
// kept for backward compatibility but ignored
type NtfyConfig struct {}

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
