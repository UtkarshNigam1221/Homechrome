// Package geo provides lightweight normalisation helpers for geo headers
// carried on incoming requests. Centroid resolution moved out — backend
// auto-populates the city_centroids table from CloudFront viewer headers
// on first sighting (no in-process map, no allowlist).
package geo

import "strings"

// MaxCityNameLen caps the length of stored city names. Longer values
// usually indicate spoofed or malformed headers — clamped before storage
// to keep the labels JSONB small.
const MaxCityNameLen = 64

// NormalizeCity lowercases, trims, and truncates the input. Returns
// "unknown" for empty input.
func NormalizeCity(input string) string {
	s := strings.ToLower(strings.TrimSpace(input))
	if s == "" {
		return "unknown"
	}
	if len(s) > MaxCityNameLen {
		s = s[:MaxCityNameLen]
	}
	return s
}

// NormalizeCountry uppercases and trims a 2-letter ISO 3166-1 alpha-2
// country code. Returns "unknown" for empty or non-2-letter input.
func NormalizeCountry(input string) string {
	s := strings.ToUpper(strings.TrimSpace(input))
	if len(s) != 2 {
		return "unknown"
	}
	return s
}
