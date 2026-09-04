package exporters

import (
	"fmt"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/vunas/blaster/internal/metrics"
	"github.com/vunas/blaster/internal/tui/theme"
	"github.com/vunas/blaster/internal/tui/widgets"
)

type TerminalReporter struct {
	StartTime time.Time
}

func NewTerminalReporter(start time.Time) *TerminalReporter {
	return &TerminalReporter{
		StartTime: start,
	}
}

func formatBandwidth(bytes int64, durationSec float64) string {
	if durationSec <= 0 {
		return "0 B/s"
	}

	rate := float64(bytes) / durationSec
	const unit = 1024

	if rate < unit {
		return fmt.Sprintf("%.0f B/s", rate)
	}

	div, exp := float64(unit), 0
	for n := rate / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.2f %cB/s", rate/div, "KMGTPE"[exp])
}

func formatPercent(value, total int64) string {
	if total == 0 {
		return "0.0%"
	}
	return fmt.Sprintf("%.1f%%", float64(value)/float64(total)*100)
}

func printSection(title string) {
	fmt.Printf("\n%s\n", theme.TextBold(title))
}

func formatPercentile(value int64, samples int64) string {
	if samples < 2 {
		return "-"
	}
	return fmt.Sprintf("%d", value)
}

func (r *TerminalReporter) Export(
	engine *metrics.EngineMetrics,
	assert *metrics.AssertionManager,
) error {
	totalReqs, iters, success, netFail, logicFail, _, dataSent, dataRecv := engine.Snapshot()

	duration := time.Since(r.StartTime)
	durationSec := duration.Seconds()

	var rps float64
	if durationSec > 0 {
		rps = float64(totalReqs) / durationSec
	}

	fmt.Println()
	title := theme.TextBold("BLASTER TEST RESULTS")
	timestamp := r.StartTime.Format("2006-01-02 15:04:05")
	fmt.Printf("%-40s%s\n", title, timestamp)

	printSection("SYSTEM")
	systemTable := widgets.NewTable("Metric", "Value")
	systemTable.AddRow("Duration", duration.Round(time.Millisecond).String())
	systemTable.AddRow("Throughput", fmt.Sprintf("%.2f req/s", rps))
	systemTable.AddRow(
		"Bandwidth",
		fmt.Sprintf(
			"%s IN · %s OUT",
			theme.TextGreen(formatBandwidth(dataRecv, durationSec)),
			theme.TextYellow(formatBandwidth(dataSent, durationSec)),
		),
	)
	iterP50, iterP90, iterP95, iterP99, iterMax := "-", "-", "-", "-", "-"
	if iters > 0 {
		iterP50 = formatPercentile(engine.IterationDuration.Percentile(50), iters)
		iterP90 = formatPercentile(engine.IterationDuration.Percentile(90), iters)
		iterP95 = formatPercentile(engine.IterationDuration.Percentile(95), iters)
		iterP99 = formatPercentile(engine.IterationDuration.Percentile(99), iters)
		iterMax = fmt.Sprintf("%d", engine.IterationDuration.Max())
	}
	systemTable.AddRow(
		"Iterations",
		fmt.Sprintf(
			"%d · p50 %sms · p90 %sms · p95 %sms · p99 %sms · max %sms",
			iters,
			iterP50,
			iterP90,
			iterP95,
			iterP99,
			iterMax,
		),
	)
	fmt.Print(systemTable.Render())

	printSection("TRAFFIC")
	totalExecutions := totalReqs + logicFail
	trafficTable := widgets.NewTable("Metric", "Count", "Rate")
	trafficTable.AddRow("Total Executions", fmt.Sprint(totalExecutions), "100.0%")
	successStr := fmt.Sprint(success)
	if success > 0 {
		successStr = theme.TextGreen(successStr)
	}
	trafficTable.AddRow("Successful Requests", successStr, formatPercent(success, totalExecutions))
	networkFailStr := fmt.Sprint(netFail)
	if netFail > 0 {
		networkFailStr = theme.TextYellow(networkFailStr)
	}
	trafficTable.AddRow("HTTP / Network Errors", networkFailStr, formatPercent(netFail, totalExecutions))
	logicFailStr := fmt.Sprint(logicFail)
	if logicFail > 0 {
		logicFailStr = theme.TextRed(logicFailStr)
	}
	trafficTable.AddRow("Assertion / Logic Fails", logicFailStr, formatPercent(logicFail, totalExecutions))
	fmt.Print(trafficTable.Render())

	statusCounts := make(map[int]int64)
	for _, node := range engine.GetAllNodes() {
		for code, count := range node.StatusCodes {
			if count > 0 {
				statusCounts[code] += count
			}
		}
	}
	if len(statusCounts) > 0 {
		printSection(fmt.Sprintf("STATUS CODES (%d Total)", totalReqs))
		var codes []int
		for code := range statusCounts {
			codes = append(codes, code)
		}
		sort.Ints(codes)

		var parts []string
		for _, code := range codes {
			count := statusCounts[code]
			codeStr := fmt.Sprint(code)

			switch {
			case code >= 200 && code < 300:
				codeStr = theme.TextGreen(codeStr)
			case code >= 400 && code < 500:
				codeStr = theme.TextYellow(codeStr)
			case code >= 500:
				codeStr = theme.TextRed(codeStr)
			}

			parts = append(parts, fmt.Sprintf("[%s]: %d", codeStr, count))
		}

		fmt.Println(strings.Join(parts, "   "))
	}

	printSection("REQUEST TIMINGS · ms")
	nodeMetrics := engine.GetAllNodes()
	var nodeIDs []string
	for nodeID := range nodeMetrics {
		nodeIDs = append(nodeIDs, nodeID)
	}
	sort.Slice(nodeIDs, func(i, j int) bool {
		extractNum := func(s string) int {
			parts := strings.Split(s, "_")
			if len(parts) == 2 {
				var num int
				fmt.Sscanf(parts[1], "%d", &num)
				return num
			}
			return 0
		}

		numI := extractNum(nodeIDs[i])
		numJ := extractNum(nodeIDs[j])

		if numI != numJ {
			return numI < numJ
		}
		return nodeIDs[i] < nodeIDs[j]
	})
	nodeTable := widgets.NewTable("Endpoint / Node", "Req", "Fail", "Min", "Avg", "p90", "p95", "p99", "Max", "TTFB")
	for _, nodeID := range nodeIDs {
		nodeMet := nodeMetrics[nodeID]
		parsedPath := extractPath(nodeMet.Name)

		var displayName string
		if nodeMet.Method != "" && parsedPath != "" {
			displayName = fmt.Sprintf("%-6s %s [%s]", nodeMet.Method, parsedPath, nodeID)
		} else if parsedPath != "" {
			displayName = fmt.Sprintf("%s [%s]", parsedPath, nodeID)
		} else {
			displayName = nodeID
		}
		reqs := atomic.LoadInt64(&nodeMet.TotalRequests)
		if reqs == 0 {
			continue
		}
		fails := atomic.LoadInt64(&nodeMet.FailCount)
		totalLatencyUs := atomic.LoadInt64(&nodeMet.TotalLatencyUs)
		avgMs := float64(totalLatencyUs) / float64(reqs) / 1000.0
		failStr := fmt.Sprint(fails)
		if fails > 0 {
			failStr = theme.TextRed(failStr)
		}
		min := "-"
		max := "-"
		if reqs > 0 {
			min = fmt.Sprint(nodeMet.ReqDuration.Min())
			max = fmt.Sprint(nodeMet.ReqDuration.Max())
		}
		p90 := formatPercentile(nodeMet.ReqDuration.Percentile(90), reqs)
		p95 := formatPercentile(nodeMet.ReqDuration.Percentile(95), reqs)
		p99 := formatPercentile(nodeMet.ReqDuration.Percentile(99), reqs)
		ttfb := formatPercentile(nodeMet.TTFB.Percentile(90), reqs)
		nodeTable.AddRow(displayName, fmt.Sprint(reqs), failStr, min, fmt.Sprintf("%.1f", avgMs), p90, p95, p99, max, ttfb)
	}
	fmt.Print(nodeTable.Render())

	branches := engine.GetAllBranches()
	loops := engine.GetAllLoops()
	polls := engine.GetAllPolls()
	matches := engine.GetAllMatches()
	hasFlow := len(branches) > 0 || len(loops) > 0 || len(polls) > 0 || len(matches) > 0
	if hasFlow {
		printSection("CONTROL FLOW ANALYTICS")
		flowTable := widgets.NewTable("ID", "Type", "Execution")

		// Branches
		branchIDs := make([]string, 0, len(branches))
		for id := range branches {
			branchIDs = append(branchIDs, id)
		}
		sort.Strings(branchIDs)
		for _, id := range branchIDs {
			b := branches[id]
			trueCount := atomic.LoadInt64(&b.TrueCount)
			falseCount := atomic.LoadInt64(&b.FalseCount)
			execution := fmt.Sprintf("true %s · false %s", theme.TextGreen(fmt.Sprint(trueCount)), theme.TextRed(fmt.Sprint(falseCount)))
			flowTable.AddRow(id, theme.TextYellow("BRANCH"), execution)
		}

		// Loops
		loopIDs := make([]string, 0, len(loops))
		for id := range loops {
			loopIDs = append(loopIDs, id)
		}
		sort.Strings(loopIDs)
		for _, id := range loopIDs {
			l := loops[id]
			entered := atomic.LoadInt64(&l.TotalEntered)
			iterations := atomic.LoadInt64(&l.TotalIterations)
			execution := fmt.Sprintf("entered %d · iterations %d", entered, iterations)
			flowTable.AddRow(id, theme.TextBlue("LOOP"), execution)
		}

		// Polls
		pollIDs := make([]string, 0, len(polls))
		for id := range polls {
			pollIDs = append(pollIDs, id)
		}
		sort.Strings(pollIDs)
		for _, id := range pollIDs {
			p := polls[id]
			entered := atomic.LoadInt64(&p.TotalEntered)
			successCount := atomic.LoadInt64(&p.SuccessCount)
			exhausted := atomic.LoadInt64(&p.Exhausted)
			execution := fmt.Sprintf("entered %d · success %s · exhausted %s", entered, theme.TextGreen(fmt.Sprint(successCount)), theme.TextRed(fmt.Sprint(exhausted)))
			flowTable.AddRow(id, theme.TextCyan("POLL"), execution)
		}

		// Matches
		matchIDs := make([]string, 0, len(matches))
		for id := range matches {
			matchIDs = append(matchIDs, id)
		}
		sort.Strings(matchIDs)
		for _, id := range matchIDs {
			m := matches[id]
			cases := m.GetAll()
			caseNames := make([]string, 0, len(cases))
			for name := range cases {
				caseNames = append(caseNames, name)
			}
			sort.Strings(caseNames)
			var parts []string
			for _, name := range caseNames {
				parts = append(parts, fmt.Sprintf("%s %d", name, cases[name]))
			}
			execution := strings.Join(parts, " · ")
			if execution == "" {
				execution = "-"
			}
			flowTable.AddRow(id, theme.TextMagenta("MATCH"), execution)
		}

		fmt.Print(flowTable.Render())
	}

	customMap := engine.Custom.GetAll()
	if len(customMap) > 0 {
		printSection("CUSTOM METRICS")
		customTable := widgets.NewTable("Metric", "Type", "Count", "Last", "Min", "Max")
		var keys []string
		for key := range customMap {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			c := customMap[key]
			last, min, max := "-", "-", "-"

			switch c.Type {
			case metrics.TypeTrend:
				last = fmt.Sprintf("%.2f", c.Last)
				min = fmt.Sprintf("%.2f", c.Min)
				max = fmt.Sprintf("%.2f", c.Max)
			case metrics.TypeCounter:
				last = fmt.Sprintf("%.2f", c.Sum)
			case metrics.TypeGauge:
				last = fmt.Sprintf("%.2f", c.Last)
			}
			customTable.AddRow(key, string(c.Type), fmt.Sprint(c.Count), last, min, max)
		}
		fmt.Print(customTable.Render())
	}

	fmt.Println()
	if err := assert.EvaluateThresholds(engine); err != nil {
		fmt.Printf("%s %s\n", theme.TextRed("✗ Thresholds failed"), err)
	} else {
		fmt.Println(theme.TextGreen("✓ All thresholds passed"))
	}
	fmt.Println()

	return nil
}
