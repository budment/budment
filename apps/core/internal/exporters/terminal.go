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
	if samples < 1 {
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
	title := theme.TextBold("BLASTER TEST SUMMARY")
	timestamp := r.StartTime.Format("2006-01-02 15:04:05")
	fmt.Printf("%-40s%s\n", title, timestamp)

	printSection("SYSTEM PERFORMANCE")
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

	if totalReqs > 0 && engine.RequestDuration != nil {
		reqP50 := formatPercentile(engine.RequestDuration.Percentile(50), totalReqs)
		reqP90 := formatPercentile(engine.RequestDuration.Percentile(90), totalReqs)
		reqP95 := formatPercentile(engine.RequestDuration.Percentile(95), totalReqs)
		reqP99 := formatPercentile(engine.RequestDuration.Percentile(99), totalReqs)
		reqMax := fmt.Sprintf("%d", engine.RequestDuration.Max())
		systemTable.AddRow(
			"Req Latency",
			fmt.Sprintf("p50 %sms · p90 %sms · p95 %sms · p99 %sms · max %sms", reqP50, reqP90, reqP95, reqP99, reqMax),
		)
	}

	// Iteration execution time.
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

	printSection("TRAFFIC BREAKDOWN")
	totalExecutions := totalReqs + logicFail
	trafficTable := widgets.NewTable("Category", "Count", "Ratio")
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
	trafficTable.AddRow("Network / HTTP Fails", networkFailStr, formatPercent(netFail, totalExecutions))
	logicFailStr := fmt.Sprint(logicFail)
	if logicFail > 0 {
		logicFailStr = theme.TextRed(logicFailStr)
	}
	trafficTable.AddRow("Logic / Assertion Fails", logicFailStr, formatPercent(logicFail, totalExecutions))
	fmt.Print(trafficTable.Render())

	topErrors := engine.GetTopErrors()
	if len(topErrors) > 0 {
		printSection("TOP SYSTEM & LOGIC ERRORS")
		errTable := widgets.NewTable("Count", "Error Message")
		type errItem struct {
			msg   string
			count int
		}
		var errList []errItem
		for msg, count := range topErrors {
			errList = append(errList, errItem{msg: msg, count: count})
		}
		sort.Slice(errList, func(i, j int) bool {
			return errList[i].count > errList[j].count
		})

		maxDisplay := min(len(errList), 5)
		for i := range maxDisplay {
			errTable.AddRow(theme.TextRed(fmt.Sprintf("%dx", errList[i].count)), errList[i].msg)
		}
		fmt.Print(errTable.Render())
	}

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
				_, _ = fmt.Sscanf(parts[1], "%d", &num)
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
		min := fmt.Sprint(nodeMet.ReqDuration.Min())
		max := fmt.Sprint(nodeMet.ReqDuration.Max())
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
	scripts := engine.GetAllScripts()

	hasFlow := len(branches) > 0 || len(loops) > 0 || len(polls) > 0 || len(matches) > 0 || len(scripts) > 0
	if hasFlow {
		printSection("CONTROL FLOW ANALYTICS")
		flowTable := widgets.NewTable("Node ID", "Type", "Execution Summary")

		// Scripts
		var scriptIDs []string
		for id := range scripts {
			scriptIDs = append(scriptIDs, id)
		}
		sort.Strings(scriptIDs)
		for _, id := range scriptIDs {
			s := scripts[id]
			calls := atomic.LoadInt64(&s.TotalCalls)
			fails := atomic.LoadInt64(&s.FailCount)
			totalLat := atomic.LoadInt64(&s.TotalLatencyUs)
			avgMs := 0.0
			if calls > 0 {
				avgMs = (float64(totalLat) / float64(calls)) / 1000.0
			}
			failText := fmt.Sprint(fails)
			if fails > 0 {
				failText = theme.TextRed(failText)
			}
			flowTable.AddRow(id, theme.TextGreen("SCRIPT"), fmt.Sprintf("calls %d · fails %s · avg %.1f ms", calls, failText, avgMs))
		}

		// Branches
		var branchIDs []string
		for id := range branches {
			branchIDs = append(branchIDs, id)
		}
		sort.Strings(branchIDs)
		for _, id := range branchIDs {
			b := branches[id]
			trueCount := atomic.LoadInt64(&b.TrueCount)
			falseCount := atomic.LoadInt64(&b.FalseCount)
			flowTable.AddRow(id, theme.TextYellow("BRANCH"), fmt.Sprintf("true %s · false %s", theme.TextGreen(fmt.Sprint(trueCount)), theme.TextRed(fmt.Sprint(falseCount))))
		}

		// Loops
		var loopIDs []string
		for id := range loops {
			loopIDs = append(loopIDs, id)
		}
		sort.Strings(loopIDs)
		for _, id := range loopIDs {
			l := loops[id]
			entered := atomic.LoadInt64(&l.TotalEntered)
			iterations := atomic.LoadInt64(&l.TotalIterations)
			flowTable.AddRow(id, theme.TextBlue("LOOP"), fmt.Sprintf("entered %d · iterations %d", entered, iterations))
		}

		// Polls
		var pollIDs []string
		for id := range polls {
			pollIDs = append(pollIDs, id)
		}
		sort.Strings(pollIDs)
		for _, id := range pollIDs {
			p := polls[id]
			entered := atomic.LoadInt64(&p.TotalEntered)
			successCount := atomic.LoadInt64(&p.SuccessCount)
			exhausted := atomic.LoadInt64(&p.Exhausted)
			flowTable.AddRow(id, theme.TextCyan("POLL"), fmt.Sprintf("entered %d · success %s · exhausted %s", entered, theme.TextGreen(fmt.Sprint(successCount)), theme.TextRed(fmt.Sprint(exhausted))))
		}

		// Matches
		var matchIDs []string
		for id := range matches {
			matchIDs = append(matchIDs, id)
		}
		sort.Strings(matchIDs)
		for _, id := range matchIDs {
			m := matches[id]
			cases := m.GetAll()
			var parts []string
			for k, v := range cases {
				parts = append(parts, fmt.Sprintf("%s: %d", k, v))
			}
			flowTable.AddRow(id, theme.TextMagenta("MATCH"), strings.Join(parts, " · "))
		}

		fmt.Print(flowTable.Render())
	}

	customMap := engine.Custom.GetAll()
	if len(customMap) > 0 {
		printSection("CUSTOM BUSINESS METRICS")
		customTable := widgets.NewTable("Metric", "Type", "Count", "Value (Last / Min / Max / Sum)")
		var keys []string
		for key := range customMap {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			c := customMap[key]
			valSummary := "-"
			switch c.Type {
			case metrics.TypeTrend:
				valSummary = fmt.Sprintf("Last: %.2f · Min: %.2f · Max: %.2f", c.Last, c.Min, c.Max)
			case metrics.TypeCounter:
				valSummary = fmt.Sprintf("Sum: %.2f", c.Sum)
			case metrics.TypeGauge:
				valSummary = fmt.Sprintf("Current: %.2f", c.Last)
			}
			customTable.AddRow(key, string(c.Type), fmt.Sprint(c.Count), valSummary)
		}
		fmt.Print(customTable.Render())
	}

	if assert != nil {
		_ = assert.EvaluateThresholds(engine)
		results := assert.GetResults()

		if len(results) > 0 {
			printSection("SLA THRESHOLDS SCORECARD")
			threshTable := widgets.NewTable("SLA Metric", "Requirement", "Actual", "Status", "Details")
			hasFailures := false

			for _, r := range results {
				statusStr := theme.TextGreen("PASS")
				actualStr := fmt.Sprintf("%.2f", r.Actual)

				if !r.Passed {
					hasFailures = true
					statusStr = theme.TextRed("FAIL")
					actualStr = theme.TextRed(actualStr)
				}

				threshTable.AddRow(r.Metric, r.Criteria, actualStr, statusStr, r.Reason)
			}
			fmt.Print(threshTable.Render())

			fmt.Println()
			if hasFailures {
				fmt.Printf("%s %s", theme.TextRed("✗ BUILD FAILED:"), "One or more SLA thresholds were breached.")
			} else {
				fmt.Printf("%s %s", theme.TextGreen("✓ BUILD PASSED:"), "All SLA criteria successfully satisfied.")
			}
		}
		if len(results) == 0 {
			fmt.Printf("\n%s %s", theme.TextDim("• SLA THRESHOLDS:"), theme.TextDim("No thresholds configured in test script."))
		}
	}
	fmt.Println()

	return nil
}
