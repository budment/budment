package widgets

import (
	"fmt"
	"math"
	"strings"

	"github.com/vunas/blaster/internal/tui/theme"
)

func RenderProgress(percent float64, width int) string {
	if width < 5 {
		width = 5
	}
	percent = math.Max(0, math.Min(100, percent))
	barBodyLen := width - 1

	filled := int((percent / 100.0) * float64(barBodyLen))
	empty := barBodyLen - filled

	var sb strings.Builder
	sb.Grow(width + 30)

	if filled > 0 {
		sb.WriteString(theme.TextCyan(strings.Repeat("=", filled)))
	}
	if empty > 0 {
		sb.WriteString(theme.TextDim(strings.Repeat("-", empty)))
	}
	sb.WriteString(theme.TextDim(">"))

	return sb.String()
}

// Returns a progress bar appended with a formatted percentage and custom label.
func RenderLabeledProgress(percent float64, width int, label string) string {
	return fmt.Sprintf("%s  %3.0f%%  %s", RenderProgress(percent, width), math.Min(100, math.Max(0, percent)), label)
}
