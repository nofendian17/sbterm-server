package log

import (
	"fmt"
	"log/slog"
	"strings"
)

func ParseLevel(s string) (Level, error) {
	var lv slog.Level
	if err := lv.UnmarshalText([]byte(s)); err != nil {
		return 0, fmt.Errorf("log: invalid level %q: %w", s, err)
	}
	return lv, nil
}

func ParseFormat(s string) (Format, error) {
	switch strings.ToLower(s) {
	case "text":
		return FormatText, nil
	case "json":
		return FormatJSON, nil
	default:
		return "", fmt.Errorf("log: invalid format %q (want text or json)", s)
	}
}
