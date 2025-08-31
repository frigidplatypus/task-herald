package taskwarrior

import (
	"testing"
)

func TestExtractTags_UnicodeAndEdgeCases(t *testing.T) {
	tests := []struct {
		desc string
		tags []string
	}{
		{"Unicode +fooß +bar-ß +qux_ü", []string{"foo", "bar-", "qux_"}}, // Only ASCII allowed, so ß/ü not matched
		{"+foo+bar+qux", []string{"foo", "bar", "qux"}},
		{"+foo-bar--baz", []string{"foo-bar--baz"}},
		{"+foo_bar-baz", []string{"foo_bar-baz"}},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got := ExtractTags(tt.desc)
			if len(got) != len(tt.tags) {
				t.Errorf("ExtractTags(%q) = %v, want %v", tt.desc, got, tt.tags)
				return
			}
			for i := range got {
				if got[i] != tt.tags[i] {
					t.Errorf("ExtractTags(%q)[%d] = %q, want %q", tt.desc, i, got[i], tt.tags[i])
				}
			}
		})
	}
}

func TestRemoveTagsFromDescription_Whitespace(t *testing.T) {
	tests := []struct {
		in  string
		out string
	}{
		{"This   is   a   +foo   task", "This is a task"},
		{"Multiple\n+foo\t+bar-baz\n+qux_1\ttags", "Multiple tags"},
		{"+foo+bar+qux", ""},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got := RemoveTagsFromDescription(tt.in)
			if got != tt.out {
				t.Errorf("RemoveTagsFromDescription(%q) = %q, want %q", tt.in, got, tt.out)
			}
		})
	}
}
