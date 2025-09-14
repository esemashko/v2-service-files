package utils

import (
	"regexp"
	"strings"
)

// GenerateCodeFromString converts a string to a safe ASCII code-like slug (used for filenames/keys)
// Very lightweight replacement to avoid dependency loss.
func GenerateCodeFromString(s string) string {
	if s == "" {
		return "code"
	}
	// Lowercase and replace spaces
	res := strings.ToLower(s)
	res = strings.TrimSpace(res)
	// Transliterate Cyrillic to Latin (basic)
	cyr := "абвгдеёжзийклмнопрстуфхцчшщыэюя"
	lat := []string{"a", "b", "v", "g", "d", "e", "e", "zh", "z", "i", "y", "k", "l", "m", "n", "o", "p", "r", "s", "t", "u", "f", "h", "c", "ch", "sh", "sch", "y", "e", "yu", "ya"}
	for i, r := range cyr {
		if i < len(lat) {
			res = strings.ReplaceAll(res, string(r), lat[i])
		}
	}
	// Remove invalid chars, keep [a-z0-9-_]
	res = regexp.MustCompile(`[^a-z0-9\-_]+`).ReplaceAllString(res, "-")
	res = strings.Trim(res, "-")
	if res == "" {
		return "code"
	}
	return res
}
