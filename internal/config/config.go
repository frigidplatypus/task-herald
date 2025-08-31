package config

import (
	"os"
	"sync"
	"time"
	"bytes"

	"gopkg.in/yaml.v3"
)

type Config struct {
	PollInterval        time.Duration `yaml:"poll_interval"`
	SyncInterval        time.Duration `yaml:"sync_interval"`
	Ntfy                NtfyConfig    `yaml:"ntfy"`
	HTTP                HTTPConfig    `yaml:"http"`
	LogLevel            string        `yaml:"log_level"`
	NotificationMessage string        `yaml:"notification_message"`
	UDAMap              UDAMap        `yaml:"udas"`
}

type NtfyConfig struct {
	URL            string            `yaml:"url"`
	Topic          string            `yaml:"topic"`
	TopicFile      string            `yaml:"topic_file"`
	Token          string            `yaml:"token"`
	Headers        map[string]string `yaml:"headers"`
	ActionsEnabled bool              `yaml:"actions_enabled"`
}

type HTTPConfig struct {
	Addr      string `yaml:"addr"`
	TLSCert   string `yaml:"tls_cert"`
	TLSKey    string `yaml:"tls_key"`
	AuthToken string `yaml:"auth_token"`
	AuthTokenFile string `yaml:"auth_token_file"`
	TLSCertFile string `yaml:"tls_cert_file"`
	TLSKeyFile string `yaml:"tls_key_file"`
	Debug     bool   `yaml:"debug"`
}

// GetTopic returns the topic, reading from file if TopicFile is set
func (n *NtfyConfig) GetTopic() string {
       if n.TopicFile != "" {
	       data, err := os.ReadFile(n.TopicFile)
	       if err == nil {
		       return string(bytes.TrimSpace(data))
	       }
       }
       return n.Topic
}

type UDAMap struct {
	NotificationDate string `yaml:"notification_date"`
	RepeatEnable     string `yaml:"repeat_enable"`
	RepeatDelay      string `yaml:"repeat_delay"`
}

// WebConfig struct removed

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
