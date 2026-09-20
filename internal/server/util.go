package server

import (
	"fmt"
	"strings"
	"time"
)

func sprintf(format string, a ...any) string { return fmt.Sprintf(format, a...) }

func unixNow() int64 { return time.Now().Unix() }

// clamp clamps v into [lo, hi].
func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// timeAgo renders a unix timestamp relative to now, BBS-style.
func timeAgo(now, t int64) string {
	d := now - t
	switch {
	case d < 5:
		return "just now"
	case d < 60:
		return fmt.Sprintf("%ds ago", d)
	case d < 3600:
		return fmt.Sprintf("%dm ago", d/60)
	case d < 86400:
		return fmt.Sprintf("%dh ago", d/3600)
	case d < 86400*30:
		return fmt.Sprintf("%dd ago", d/86400)
	default:
		return time.Unix(t, 0).Format("2006-01-02")
	}
}

func dateOf(t int64) string { return time.Unix(t, 0).Format("2006-01-02 15:04") }

// isValidUsername enforces the registration rules: 3-32 characters, allowing
// letters, digits, underscores and hyphens.
func isValidUsername(s string) bool {
	if len(s) < 3 || len(s) > 32 {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '_', r == '-':
		default:
			return false
		}
	}
	return true
}

func lower(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "\u2026"
}