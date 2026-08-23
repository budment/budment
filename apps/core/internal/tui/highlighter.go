package tui

import (
	"regexp"
	"strings"

	"github.com/vunas/blaster/internal/tui/theme"
)

// Applies ANSI color codes to raw JSON strings for terminal output.
func HighlightJSON(jsonStr string) string {
	if !theme.ColorEnabled {
		return jsonStr
	}

	keyRegex := regexp.MustCompile(`"([^"]+)":`)
	jsonStr = keyRegex.ReplaceAllStringFunc(jsonStr, func(match string) string {
		key := match[1 : len(match)-2]
		return theme.TextBlue(`"`+key+`"`) + ":"
	})

	stringRegex := regexp.MustCompile(`:\s*"([^"]*)"`)
	jsonStr = stringRegex.ReplaceAllStringFunc(jsonStr, func(match string) string {
		val := match[strings.Index(match, `"`):]
		return `: ` + theme.TextGreen(val)
	})

	numberRegex := regexp.MustCompile(`:\s*([0-9.]+)`)
	jsonStr = numberRegex.ReplaceAllStringFunc(jsonStr, func(match string) string {
		val := strings.TrimSpace(match[strings.Index(match, ":")+1:])
		return `: ` + theme.TextYellow(val)
	})

	boolRegex := regexp.MustCompile(`:\s*(true|false|null)`)
	jsonStr = boolRegex.ReplaceAllStringFunc(jsonStr, func(match string) string {
		val := strings.TrimSpace(match[1:])
		return `: ` + theme.TextMagenta(val)
	})

	return jsonStr
}
