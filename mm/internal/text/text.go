// Package text formats the plain strings the pickers lay out in columns.
package text

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode"

	"github.com/charmbracelet/x/ansi"
)

// Width is the number of terminal columns s takes, so CJK and emoji count
// double.
func Width(s string) int {
	return ansi.StringWidth(s)
}

// Fit truncates s to w columns, ending it with … when cut, and pads it to
// exactly w.
func Fit(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if Width(s) > w {
		s = ansi.Truncate(s, w, "…")
	}
	return s + strings.Repeat(" ", max(w-Width(s), 0))
}

// PadLeft right-aligns s in w columns.
func PadLeft(s string, w int) string {
	return strings.Repeat(" ", max(w-Width(s), 0)) + s
}

// Age renders seconds as their largest whole unit: 42s, 5m, 3h, 2d, 4mo, 1y.
func Age(s int64) string {
	s = max(s, 0)
	switch {
	case s < 60:
		return fmt.Sprintf("%ds", s)
	case s < 3600:
		return fmt.Sprintf("%dm", s/60)
	case s < 86400:
		return fmt.Sprintf("%dh", s/3600)
	case s < 2592000:
		return fmt.Sprintf("%dd", s/86400)
	case s < 31536000:
		return fmt.Sprintf("%dmo", s/2592000)
	default:
		return fmt.Sprintf("%dy", s/31536000)
	}
}

// ParseDuration reads zellij's own phrasing, like "3days 1h 24m 8s", as
// seconds. Tokens that don't start with a number are skipped.
func ParseDuration(s string) int64 {
	var total int64
	for tok := range strings.FieldsSeq(s) {
		i := strings.IndexFunc(tok, func(r rune) bool { return !unicode.IsDigit(r) })
		if i == 0 {
			continue
		}
		if i < 0 {
			i = len(tok)
		}
		n, err := strconv.ParseInt(tok[:i], 10, 64)
		if err != nil {
			continue
		}
		switch tok[i:] {
		case "s", "sec", "secs":
			total += n
		case "m", "min", "mins":
			total += n * 60
		case "h", "hour", "hours":
			total += n * 3600
		case "d", "day", "days":
			total += n * 86400
		case "w", "week", "weeks":
			total += n * 604800
		case "month", "months":
			total += n * 2592000
		case "y", "year", "years":
			total += n * 31536000
		}
	}
	return total
}

// Tilde abbreviates a path under $HOME to ~.
func Tilde(path string) string {
	home := os.Getenv("HOME")
	if home != "" && (path == home || strings.HasPrefix(path, home+"/")) {
		return "~" + path[len(home):]
	}
	return path
}
