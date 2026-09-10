package widgets

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/budment/budment/internal/tui/theme"
)

var ansiRegexp = regexp.MustCompile(`\x1b\[[0-9;]*m`)

var numericHeaders = map[string]struct{}{
	"count": {}, "rate": {}, "req": {}, "fail": {},
	"min": {}, "avg": {}, "p50": {}, "p90": {},
	"p95": {}, "p99": {}, "max": {}, "ttfb": {},
}

const maxTerminalWidth = 120

type Table struct {
	Headers []string
	Rows    [][]string
}

func NewTable(headers ...string) *Table {
	return &Table{Headers: headers}
}

func (t *Table) AddRow(row ...string) {
	t.Rows = append(t.Rows, row)
}

func visibleWidth(s string) int {
	return utf8.RuneCountInString(ansiRegexp.ReplaceAllString(s, ""))
}

func padRight(s string, width int) string {
	if pad := width - visibleWidth(s); pad > 0 {
		return s + strings.Repeat(" ", pad)
	}
	return s
}

func padLeft(s string, width int) string {
	if pad := width - visibleWidth(s); pad > 0 {
		return strings.Repeat(" ", pad) + s
	}
	return s
}

func isNumericColumn(header string) bool {
	_, ok := numericHeaders[strings.ToLower(header)]
	return ok
}

func (t *Table) calculateColumnWidths() ([]int, int) {
	if len(t.Headers) == 0 {
		return nil, 0
	}

	widths := make([]int, len(t.Headers))
	for i, header := range t.Headers {
		widths[i] = visibleWidth(header)
	}

	for _, row := range t.Rows {
		for i, col := range row {
			if i < len(widths) {
				widths[i] = max(widths[i], visibleWidth(col))
			}
		}
	}

	totalWidth := 0
	for i, w := range widths {
		totalWidth += w
		if i < len(widths)-1 {
			totalWidth += 2
		}
	}

	return widths, totalWidth
}

func (t *Table) Render() string {
	colWidths, tableWidth := t.calculateColumnWidths()
	if len(colWidths) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.Grow((len(t.Rows) + 3) * (tableWidth + 10))

	for i, header := range t.Headers {
		sb.WriteString(padRight(theme.TextBold(header), colWidths[i]))
		if i < len(t.Headers)-1 {
			sb.WriteString("  ")
		}
	}
	sb.WriteByte('\n')

	separatorWidth := tableWidth
	if separatorWidth > maxTerminalWidth {
		separatorWidth = maxTerminalWidth
	}
	sb.WriteString(strings.Repeat("-", separatorWidth))
	sb.WriteByte('\n')

	for _, row := range t.Rows {
		for i := range t.Headers {
			var col string
			if i < len(row) {
				col = row[i]
			}

			if i == len(t.Headers)-1 {
				sb.WriteString(col)
				continue
			}

			if isNumericColumn(t.Headers[i]) {
				sb.WriteString(padLeft(col, colWidths[i]))
			} else {
				sb.WriteString(padRight(col, colWidths[i]))
			}

			if i < len(t.Headers)-1 {
				sb.WriteString("  ")
			}
		}
		sb.WriteByte('\n')
	}

	return sb.String()
}

func (t *Table) Width() int {
	_, totalWidth := t.calculateColumnWidths()
	if totalWidth > maxTerminalWidth {
		return maxTerminalWidth
	}
	return totalWidth
}

func (t *Table) String() string {
	return t.Render()
}
