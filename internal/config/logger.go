package config

import (
	"fmt"
	"strings"
	"sync"
)

type LogLevel int

const (
	ERROR LogLevel = iota
	WARN
	INFO
	DEBUG
	VERBOSE
)

var (
	logLevel LogLevel = INFO
	once     sync.Once
)

func ParseLogLevel(level string) LogLevel {
	switch strings.ToLower(level) {
	case "error":
		return ERROR
	case "warn":
		return WARN
	case "info":
		return INFO
	case "debug":
		return DEBUG
	case "verbose":
		return VERBOSE
	default:
		return INFO
	}
}

func SetLogLevelFromConfig(cfg *Config) {
	once.Do(func() {
		logLevel = ParseLogLevel(cfg.LogLevel)
	})
}

func Log(level LogLevel, format string, a ...interface{}) {
	if level <= logLevel {
		fmt.Printf("[%s] ", strings.ToUpper(level.String()))
		fmt.Printf(format+"\n", a...)
	}
}

func (l LogLevel) String() string {
	switch l {
	case ERROR:
		return "error"
	case WARN:
		return "warn"
	case INFO:
		return "info"
	case DEBUG:
		return "debug"
	case VERBOSE:
		return "verbose"
	default:
		return "info"
	}
}
