package domain

import (
	"regexp"
	"strings"
)

var (
	nonWord = regexp.MustCompile(`[^а-я0-9\s]+`)
	spaces  = regexp.MustCompile(`\s+`)
)

func Normalize(text string) string {
	text = strings.ToLower(text)
	text = strings.ReplaceAll(text, "ё", "е")
	text = nonWord.ReplaceAllString(text, " ")
	text = spaces.ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}

func NormalizeLocations(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		n := Normalize(part)
		if n == "" {
			continue
		}
		out = append(out, n)
	}
	return out
}
