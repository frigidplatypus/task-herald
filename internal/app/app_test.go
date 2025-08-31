package app

import (
	"task-herald/internal/config"
	"task-herald/internal/taskwarrior"
	"testing"
)

func TestGetUDA_ReturnsNotificationDate(t *testing.T) {
	cfg := &config.Config{UDAMap: config.UDAMap{NotificationDate: "notification_date"}}
	config.Set(cfg)

	task := taskwarrior.Task{NotificationDate: "2025-08-31 14:30:00"}
	v, ok := getUDA(task, "notification_date")
	if !ok {
		t.Fatalf("expected ok=true, got false")
	}
	if v != task.NotificationDate {
		t.Fatalf("expected %q, got %q", task.NotificationDate, v)
	}
}

func TestGetUDA_ReturnsFalseForEmptyNotificationDate(t *testing.T) {
	cfg := &config.Config{UDAMap: config.UDAMap{NotificationDate: "notification_date"}}
	config.Set(cfg)

	task := taskwarrior.Task{NotificationDate: ""}
	_, ok := getUDA(task, "notification_date")
	if ok {
		t.Fatalf("expected ok=false for empty notification date, got true")
	}
}

func TestGetUDA_CustomMappingFallback(t *testing.T) {
	// If UDAMap maps to a custom field, getUDA should attempt getTaskField and currently return false
	cfg := &config.Config{UDAMap: config.UDAMap{NotificationDate: "taskherald.notification_date"}}
	config.Set(cfg)

	task := taskwarrior.Task{NotificationDate: "2025-08-31 14:30:00"}
	v, ok := getUDA(task, "taskherald.notification_date")
	if ok {
		t.Fatalf("expected ok=false for custom UDA mapping (not implemented), got true with value %q", v)
	}
}
