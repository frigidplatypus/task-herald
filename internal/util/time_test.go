package util

import (
	"testing"
	"time"
)

func TestParseNotificationDate_Formats(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		layout string
	}{
		{"RFC3339", "2025-08-31T14:30:00Z", time.RFC3339},
		{"T no zone", "2025-08-31T14:30:00", "2006-01-02T15:04:05"},
		{"space no zone", "2025-08-31 14:30", "2006-01-02 15:04"},
		{"compact UTC", "20250831T143000Z", "20060102T150405Z"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseNotificationDate(tc.input)
			if err != nil {
				t.Fatalf("ParseNotificationDate(%q) error: %v", tc.input, err)
			}
			// parse expected using the same layout (assume UTC when 'Z' present)
			exp, perr := time.Parse(tc.layout, tc.input)
			if perr == nil {
				// For inputs without zone, the function may return local or exact; ensure result is non-zero and close to expected
				if exp.IsZero() {
					t.Fatalf("expected non-zero time for %q", tc.input)
				}
				// allow slight differences in location but times should match by components (year, month, day, hour, min)
				if got.Year() != exp.Year() || got.Month() != exp.Month() || got.Day() != exp.Day() || got.Hour() != exp.Hour() || got.Minute() != exp.Minute() {
					t.Fatalf("parsed time components differ: got=%v expected=%v", got, exp)
				}
			}
		})
	}
}

func TestParseNotificationDate_Invalid(t *testing.T) {
	_, err := ParseNotificationDate("not-a-date")
	if err == nil {
		t.Error("expected error for invalid date string, got nil")
	}
}

func TestParseNotificationDate_MoreCases(t *testing.T) {
	cases := []struct{
		input string
	}{
		{"2025-08-31 23:59:59"}, // seconds with space
		{"2025-08-31T23:59:59"}, // T without zone
		{"2025-08-31T23:59:59Z"}, // RFC3339 Z
		{"20250831T235959Z"},     // compact
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			if _, err := ParseNotificationDate(tc.input); err != nil {
				t.Fatalf("ParseNotificationDate(%q) error: %v", tc.input, err)
			}
		})
	}
}

func TestParseNotificationDate_TimezoneMatrix(t *testing.T) {
	// table-driven timezones and formats
	cases := []struct{ input string }{
		{"2025-08-31T12:00:00Z"},
		{"2025-08-31T12:00:00+02:00"},
		{"2025-08-31 12:00:00"},
		{"20250831T120000Z"},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			if _, err := ParseNotificationDate(tc.input); err != nil {
				t.Fatalf("ParseNotificationDate(%q) error: %v", tc.input, err)
			}
		})
	}
}
