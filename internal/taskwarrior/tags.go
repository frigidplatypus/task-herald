package taskwarrior

import (
	"regexp"
	"strings"
	"unicode"
)

// tagPattern matches +tag in descriptions (allows dashes and underscores)
var tagPattern = regexp.MustCompile(`\B\+([a-zA-Z0-9_\-]+)`)

// ExtractTags finds all +tags in a description and returns them as a slice
func ExtractTags(description string) []string {
	matches := tagPattern.FindAllStringSubmatch(description, -1)
	tags := make([]string, 0, len(matches))
	for _, m := range matches {
		tags = append(tags, m[1])
	}
	return tags
}

// RemoveTagsFromDescription strips all +tags from a description
func RemoveTagsFromDescription(description string) string {
	clean := tagPattern.ReplaceAllString(description, "")
	return strings.TrimSpace(clean)
}

// DashToCamel converts dash-separated strings to camelCase (e.g., foo-bar-baz -> fooBarBaz)
func DashToCamel(s string) string {
	var result strings.Builder
	upper := false
	for i, r := range s {
		if r == '-' {
			upper = true
			continue
		}
		if upper {
			result.WriteRune(unicode.ToUpper(r))
			upper = false
		} else if i == 0 {
			result.WriteRune(unicode.ToLower(r))
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}
