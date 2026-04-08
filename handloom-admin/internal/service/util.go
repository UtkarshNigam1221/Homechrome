package service

import (
	"regexp"
	"strings"
)

var slugDashCollapse = regexp.MustCompile("-+")

// generateSlug creates a URL-friendly slug from a name
func generateSlug(name string) string {
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	// Remove special characters
	var result strings.Builder
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}
	slug = slugDashCollapse.ReplaceAllString(result.String(), "-")
	slug = strings.Trim(slug, "-")
	return slug
}
