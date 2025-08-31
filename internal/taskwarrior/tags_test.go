package taskwarrior

import (
	"reflect"
	"testing"
)

func TestExtractTags(t *testing.T) {
	tests := []struct {
		desc string
		tags []string
	}{
		{"This is a +foo task", []string{"foo"}},
		{"Multiple +foo +bar-baz +qux_1 tags", []string{"foo", "bar-baz", "qux_1"}},
		{"No tags here", []string{}},
		{"Edge+case+tag", []string{"case", "tag"}},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got := ExtractTags(tt.desc)
			if !reflect.DeepEqual(got, tt.tags) {
				t.Errorf("ExtractTags(%q) = %v, want %v", tt.desc, got, tt.tags)
			}
		})
	}
}

func TestRemoveTagsFromDescription(t *testing.T) {
	tests := []struct {
		in  string
		out string
	}{
		{"This is a +foo task", "This is a task"},
		{"Multiple +foo +bar-baz +qux_1 tags", "Multiple tags"},
		{"No tags here", "No tags here"},
		{"Edge+case+tag", "Edge"},
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

func TestDashToCamel(t *testing.T) {
	tests := []struct {
		in  string
		out string
	}{
		{"foo-bar", "fooBar"},
		{"foo-bar-baz", "fooBarBaz"},
		{"foobar", "foobar"},
		{"foo-bar_baz", "fooBar_baz"},
		{"foo--bar", "fooBar"},
		{"-foo-bar", "FooBar"},
		{"foo-", "foo"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got := DashToCamel(tt.in)
			if got != tt.out {
				t.Errorf("DashToCamel(%q) = %q, want %q", tt.in, got, tt.out)
			}
		})
	}
}
