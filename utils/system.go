package utils

import (
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

// ComputeFingerprintHash returns base64url(SHA-256(normalizedUA + "|" + normalizedPlatform))
// normalizedUA uses browser family + MAJOR version, OS family + MAJOR version, and device type (mobile/desktop)
func ComputeFingerprintHash(userAgent string, platform string) string {
	normUA := normalizeUserAgent(userAgent)
	pf := strings.TrimSpace(strings.ToLower(platform))
	data := normUA + "|" + pf
	sum := sha256.Sum256([]byte(data))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// ExtractPlatform tries to deduce a coarse platform string from headers
// Prefer explicit X-Platform header; fallback to User-Agent heuristics
func ExtractPlatform(r *http.Request) string {
	if r == nil {
		return ""
	}
	if v := r.Header.Get("X-Platform"); v != "" {
		return v
	}
	ua := r.Header.Get("User-Agent")
	return ExtractPlatformFromString(ua)
}

// ExtractPlatformFromString deduces platform from a User-Agent string
func ExtractPlatformFromString(ua string) string {
	l := strings.ToLower(ua)
	switch {
	case strings.Contains(l, "android"):
		return "android"
	case strings.Contains(l, "iphone") || strings.Contains(l, "ipad") || strings.Contains(l, "ios"):
		return "ios"
	case strings.Contains(l, "mac os") || strings.Contains(l, "macintosh") || strings.Contains(l, "macos"):
		return "macos"
	case strings.Contains(l, "windows"):
		return "windows"
	case strings.Contains(l, "linux"):
		return "linux"
	default:
		return "web"
	}
}

// normalizeUserAgent reduces UA to stable components (browser family + major, OS family + major, device type)
func normalizeUserAgent(ua string) string {
	l := strings.ToLower(strings.TrimSpace(ua))
	if l == "" {
		return "ua:unknown;os:unknown;d:unknown"
	}
	browser := parseBrowser(l)
	os := parseOS(l)
	device := "desktop"
	if strings.Contains(l, "mobile") {
		device = "mobile"
	}
	return "b:" + browser + ";o:" + os + ";d:" + device
}

func parseBrowser(l string) string {
	// Order matters: Edge, Chrome/Chromium, Firefox, Safari (Version/x)
	if m := regexp.MustCompile(`\bedg/(\d+)`).FindStringSubmatch(l); len(m) == 2 {
		return "edge-" + m[1]
	}
	if m := regexp.MustCompile(`\bcrios/(\d+)`).FindStringSubmatch(l); len(m) == 2 {
		return "chrome-ios-" + m[1]
	}
	if m := regexp.MustCompile(`\bchrome/(\d+)`).FindStringSubmatch(l); len(m) == 2 {
		return "chrome-" + m[1]
	}
	if m := regexp.MustCompile(`\bfirefox/(\d+)`).FindStringSubmatch(l); len(m) == 2 {
		return "firefox-" + m[1]
	}
	// Safari typically has Version/x.y Safari/...
	if strings.Contains(l, "safari/") {
		if m := regexp.MustCompile(`\bversion/(\d+)`).FindStringSubmatch(l); len(m) == 2 {
			return "safari-" + m[1]
		}
		return "safari"
	}
	return "other"
}

func parseOS(l string) string {
	if m := regexp.MustCompile(`windows nt\s+(\d+)(?:\.\d+)?`).FindStringSubmatch(l); len(m) == 2 {
		return "windows-" + m[1]
	}
	if m := regexp.MustCompile(`mac os x\s+(\d+)`).FindStringSubmatch(l); len(m) == 2 {
		return "macos-" + m[1]
	}
	if m := regexp.MustCompile(`cpu (?:iphone|ios|iphone os|ios os)?\s*os\s*(\d+)`).FindStringSubmatch(l); len(m) == 2 {
		return "ios-" + m[1]
	}
	if m := regexp.MustCompile(`android\s+(\d+)`).FindStringSubmatch(l); len(m) == 2 {
		return "android-" + m[1]
	}
	if strings.Contains(l, "linux") {
		return "linux"
	}
	if strings.Contains(l, "cros") {
		return "chromeos"
	}
	return "other"
}

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

// ExtractCleanDomain extracts host without scheme and port from a URL string (e.g., Origin header)
func ExtractCleanDomain(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	u, err := url.Parse(s)
	if err == nil && u.Host != "" {
		host := u.Host
		if h, _, ok := strings.Cut(host, ":"); ok {
			return h
		}
		return host
	}
	// Fallback: strip scheme manually
	if strings.HasPrefix(s, "http://") {
		s = strings.TrimPrefix(s, "http://")
	} else if strings.HasPrefix(s, "https://") {
		s = strings.TrimPrefix(s, "https://")
	}
	if h, _, ok := strings.Cut(s, "/"); ok {
		s = h
	}
	if h, _, ok := strings.Cut(s, ":"); ok {
		s = h
	}
	return s
}
