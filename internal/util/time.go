package util

import (
	"fmt"
	"time"
)

// ParseNotificationDate parses a date string in common Taskwarrior formats and returns it in local time
func ParseNotificationDate(s string) (time.Time, error) {
	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04",
		"20060102T150405Z", // Taskwarrior compact UTC format
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			// Only convert to local time if the parsed time is in Local
			if t.Location() == time.UTC || t.Location().String() != "Local" {
				return t, nil
			}
			return t.In(time.Local), nil
		}
	}
	return time.Time{}, fmt.Errorf("could not parse notification date: %s", s)
}
