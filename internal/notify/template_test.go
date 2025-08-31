package notify

import (
	"testing"
	"time"
)

func TestRenderMessage_DefaultAndTemplate(t *testing.T) {
	now := time.Date(2025, 8, 31, 15, 0, 0, 0, time.UTC)
	task := TaskInfo{
		ID:               "1",
		UUID:             "u1",
		Description:      "do things",
		Tags:             []string{"a", "b"},
		Due:              &now,
		NotificationDate: &now,
		Project:          "proj",
		Priority:         "H",
	}

	// default template should include description and project
	msg, err := RenderMessage(task, "")
	if err != nil {
		t.Fatalf("RenderMessage default failed: %v", err)
	}
	if len(msg) == 0 {
		t.Fatalf("expected non-empty default message")
	}

	// custom template referencing fields and tags
	tmpl := "{{.Description}}|{{.ID}}|{{.Project}}|{{range .Tags}}{{.}};{{end}}|{{if .Due}}{{.Due.Format \"2006-01-02\"}}{{end}}"
	msg2, err := RenderMessage(task, tmpl)
	if err != nil {
		t.Fatalf("RenderMessage custom failed: %v", err)
	}
	if msg2 != "do things|1|proj|a;b;|2025-08-31" {
		t.Fatalf("unexpected rendered message: %q", msg2)
	}
}

func TestRenderMessage_EmptyFields(t *testing.T) {
	task := TaskInfo{
		ID:          "2",
		UUID:        "u2",
		Description: "something",
		Tags:        []string{},
		Project:     "",
	}
	msg, err := RenderMessage(task, "{{.Project}}|{{range .Tags}}{{.}}{{else}}(no-tags){{end}}")
	if err != nil {
		t.Fatalf("RenderMessage empty fields failed: %v", err)
	}
	if msg != "|(no-tags)" {
		t.Fatalf("unexpected rendered message for empty fields: %q", msg)
	}
}

func TestRenderMessage_EscapedMultilineAndTagOrder(t *testing.T) {
	now := time.Date(2025, 8, 31, 16, 7, 0, 0, time.UTC)
	task := TaskInfo{
		ID:               "10",
		UUID:             "u10",
		Description:      "line1\nline2 & <special> {{braces}}",
		Tags:             []string{"b", "a"},
		NotificationDate: &now,
		Project:          "proj",
	}

	tmpl := "{{.Description}}|{{range .Tags}}{{.}};{{end}}|{{if .NotificationDate}}{{.NotificationDate.Format \"2006-01-02 15:04\"}}{{end}}"
	got, err := RenderMessage(task, tmpl)
	if err != nil {
		t.Fatalf("RenderMessage escaped failed: %v", err)
	}
	want := "line1\nline2 & <special> {{braces}}|b;a;|2025-08-31 16:07"
	if got != want {
		t.Fatalf("unexpected rendered escaped/multiline message:\n got: %q\nwant: %q", got, want)
	}
}

func TestRenderMessage_NotificationDateFormatting(t *testing.T) {
	// ensure NotificationDate formats correctly in template
	nd := time.Date(2025, 12, 25, 9, 30, 0, 0, time.FixedZone("UTC+2", 2*3600))
	task := TaskInfo{
		ID:               "20",
		Description:      "xmas",
		NotificationDate: &nd,
	}
	tmpl := "Notify at: {{if .NotificationDate}}{{.NotificationDate.Format \"2006-01-02 15:04 -0700\"}}{{end}}"
	got, err := RenderMessage(task, tmpl)
	if err != nil {
		t.Fatalf("RenderMessage notification date format failed: %v", err)
	}
	// Expect timezone offset +0200
	if got != "Notify at: 2025-12-25 09:30 +0200" {
		t.Fatalf("unexpected notification date formatting: got %q", got)
	}
}

func TestRenderMessage_MalformedTemplate(t *testing.T) {
	task := TaskInfo{ID: "1", Description: "x"}
	// missing closing brace should produce parse error
	if _, err := RenderMessage(task, "{{.Description"); err == nil {
		t.Fatalf("expected error for malformed template, got nil")
	}
}
