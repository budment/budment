package widgets

import (
	"strings"

	"github.com/vunas/blaster/internal/tui/theme"
)

var sparkChars = []string{" ", "▂", "▃", "▄", "▅", "▆", "▇", "█"}

// Converts a slice of numbers into a mini chart (e.g., ▂▃▄▅█).
func RenderSparkline(data []float64) string {
	if len(data) == 0 {
		return ""
	}

	maxVal := data[0]
	for _, v := range data {
		if v > maxVal {
			maxVal = v
		}
	}

	if maxVal == 0 {
		return strings.Repeat(" ", len(data))
	}

	var sb strings.Builder
	for _, v := range data {
		ratio := v / maxVal
		charIdx := int(ratio * 7)
		if charIdx < 0 {
			charIdx = 0
		}
		if charIdx > 7 {
			charIdx = 7
		}
		sb.WriteString(sparkChars[charIdx])
	}

	return theme.TextMagenta(sb.String())
}
