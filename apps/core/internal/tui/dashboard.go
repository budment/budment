package tui

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/budment/budment/internal/metrics"
	"github.com/budment/budment/internal/tui/theme"
	"github.com/budment/budment/internal/tui/widgets"
)

var (
	spinners   = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	ansiRegexp = regexp.MustCompile(`\x1b\[[0-9;]*m`)
)

type LiveDashboard struct {
	engine         *metrics.EngineMetrics
	startTime      time.Time
	targetDuration time.Duration
	targetVUs      int
	targetIters    int
	spinnerIdx     int
	lastLineCount  int
	LogChan        chan string
}

func NewLiveDashboard(engine *metrics.EngineMetrics, vus int, duration time.Duration, iters int) *LiveDashboard {
	return &LiveDashboard{
		engine:         engine,
		targetDuration: duration,
		targetVUs:      vus,
		targetIters:    iters,
		LogChan:        make(chan string, 10000),
	}
}

func visibleWidth(s string) int {
	return len([]rune(ansiRegexp.ReplaceAllString(s, "")))
}

func padRight(s string, width int) string {
	if padding := width - visibleWidth(s); padding > 0 {
		return s + strings.Repeat(" ", padding)
	}
	return s
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return strconv.FormatInt(b, 10) + " B"
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	if d < time.Minute {
		return fmt.Sprintf("%02ds", int(d.Seconds()))
	}
	return fmt.Sprintf("%dm %02ds", int(d/time.Minute), int((d%time.Minute)/time.Second))
}

func truncateStr(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return string(runes[:maxLen])
	}
	return string(runes[:maxLen-3]) + "..."
}

type errCountItem struct {
	msg   string
	count int
}

func (d *LiveDashboard) Start(ctx context.Context) {
	d.startTime = time.Now()
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	HideCursor()
	defer ShowCursor()

	for {
		select {
		case <-ctx.Done():
			d.flushLogs()
			d.clearFrame()
			return
		case <-ticker.C:
			d.renderFrame(false)
		case logMsg := <-d.LogChan:
			d.clearFrame()
			os.Stdout.WriteString(logMsg + "\n")
			d.renderFrame(false)
		}
	}
}

func (d *LiveDashboard) flushLogs() {
	for {
		select {
		case logMsg := <-d.LogChan:
			d.clearFrame()
			os.Stdout.WriteString(logMsg + "\n")
		default:
			return
		}
	}
}

func (d *LiveDashboard) clearFrame() {
	if d.lastLineCount > 0 {
		_, _ = fmt.Fprintf(os.Stdout, "\033[%dA\033[J", d.lastLineCount)
		d.lastLineCount = 0
	}
}

func (d *LiveDashboard) renderFrame(isFinal bool) {
	d.clearFrame()
	d.spinnerIdx = (d.spinnerIdx + 1) % len(spinners)

	total, iters, success, netFail, logicFail, activeVUs, dataSent, dataRecv := d.engine.Snapshot()
	failures := netFail + logicFail
	elapsed := time.Since(d.startTime)

	var rps float64
	if elapsed > 0 {
		rps = float64(total) / elapsed.Seconds()
	}

	var sb strings.Builder
	sb.Grow(2048)

	status := theme.Success("RUNNING")
	if isFinal {
		status = theme.TextDim("FINISHED")
		activeVUs = 0
	}
	fmt.Fprintf(&sb, "%s %sBUDMENT ENGINE%s%s%s\n\n",
		theme.TextCyan(spinners[d.spinnerIdx]),
		theme.Bold,
		theme.Reset,
		strings.Repeat(" ", 38),
		status)

	if failures > 0 {
		if errMap := d.engine.GetTopErrors(); len(errMap) > 0 {
			sortedErrs := make([]errCountItem, 0, len(errMap))
			for msg, count := range errMap {
				sortedErrs = append(sortedErrs, errCountItem{msg: msg, count: count})
			}

			slices.SortFunc(sortedErrs, func(a, b errCountItem) int {
				if a.count != b.count {
					return b.count - a.count
				}
				return strings.Compare(a.msg, b.msg)
			})

			sb.WriteString("  ")
			sb.WriteString(theme.TextRed("Top Errors"))
			sb.WriteString("\n")
			for i, item := range sortedErrs {
				if i >= 3 {
					break
				}
				fmt.Fprintf(&sb, "  %s  %s\n", theme.TextDim(truncateStr(item.msg, 65)), theme.TextRed(fmt.Sprintf("%3dx", item.count)))
			}
			sb.WriteString("\n")
		}
	}

	const leftColWidth = 28
	left := func(l, v string) string { return padRight(fmt.Sprintf("  %-12s %s", l, v), leftColWidth) }
	right := func(l, v string) string { return fmt.Sprintf("%-12s %s", l, v) }

	vUsStr := fmt.Sprintf("%s active", theme.TextCyan(fmt.Sprint(activeVUs)))
	if d.targetVUs > 0 {
		vUsStr = fmt.Sprintf("%s / %d", theme.TextCyan(fmt.Sprint(activeVUs)), d.targetVUs)
	}

	errStr := theme.TextGreen("0")
	if failures > 0 {
		errStr = theme.TextRed(fmt.Sprint(failures))
	}

	sb.WriteString(left("VUs", vUsStr))
	sb.WriteString(right("Requests", strconv.FormatInt(total, 10)))
	sb.WriteString("\n")
	sb.WriteString(left("Throughput", fmt.Sprintf("%.1f req/s", rps)))
	sb.WriteString(right("Success", strconv.FormatInt(success, 10)))
	sb.WriteString("\n")
	sb.WriteString(left("Errors", errStr))
	sb.WriteString(right("Data", fmt.Sprintf("%s ↑ · %s ↓", theme.TextYellow(formatBytes(dataSent)), theme.TextYellow(formatBytes(dataRecv)))))
	sb.WriteString("\n")

	if rpsHistory, _ := d.engine.GetHistory(); len(rpsHistory) > 0 {
		sb.WriteString("\n  Traffic\n")
		fmt.Fprintf(&sb, "  %-14s %s\n", fmt.Sprintf("%.1f req/s", rps), theme.TextCyan(widgets.RenderSparkline(rpsHistory)))
	}

	sb.WriteString("\n")
	hasDuration := d.targetDuration > 0
	hasIters := d.targetIters > 0

	if !hasDuration && !hasIters {
		fmt.Fprintf(&sb, "  %sTime%s     : %s\n", theme.Dim, theme.Reset, formatDuration(elapsed))
	} else {
		const fixedBarWidth = 60

		if hasIters {
			displayIters := min(int(iters), d.targetIters)
			percent := (float64(displayIters) / float64(d.targetIters)) * 100
			sb.WriteString("  ")
			sb.WriteString(widgets.RenderLabeledProgress(percent, fixedBarWidth, fmt.Sprintf("%d / %d iters\n", displayIters, d.targetIters)))
		}

		if hasDuration {
			percent := (elapsed.Seconds() / d.targetDuration.Seconds()) * 100
			sb.WriteString("  ")
			sb.WriteString(widgets.RenderLabeledProgress(percent, fixedBarWidth, fmt.Sprintf("%s / %s\n", formatDuration(elapsed), formatDuration(d.targetDuration))))
		}
	}

	frameContent := sb.String()
	os.Stdout.WriteString(frameContent)
	d.lastLineCount = strings.Count(frameContent, "\n")
}
