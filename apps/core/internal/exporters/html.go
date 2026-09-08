package exporters

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"html"
	"sort"
	"strings"
	"sync/atomic"
	"text/template"
	"time"

	"github.com/vunas/blaster/internal/filesystem"
	"github.com/vunas/blaster/internal/metrics"
	"github.com/vunas/blaster/internal/planner"
)

//go:embed template.html
var htmlTemplate string

type ExportScenario struct {
	Name  string
	Graph *planner.Graph
}

type ThresholdUI struct {
	Metric   string  `json:"metric"`
	Criteria string  `json:"criteria"`
	Actual   float64 `json:"actual"`
	Passed   bool    `json:"passed"`
	Reason   string  `json:"reason"`
}

type FlowUI struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Details string `json:"details"`
}

type TimeSeriesUI struct {
	Time string  `json:"time"`
	RPS  float64 `json:"rps"`
	VUs  int64   `json:"vus"`
}

type NodeUI struct {
	ID    string  `json:"id"`
	Reqs  int64   `json:"reqs"`
	Fails int64   `json:"fails"`
	Min   int64   `json:"min"`
	Avg   float64 `json:"avg"`
	P90   int64   `json:"p90"`
	P95   int64   `json:"p95"`
	P99   int64   `json:"p99"`
	Max   int64   `json:"max"`
	TTFB  int64   `json:"ttfb"`
	TCP   int64   `json:"tcp"`
	TLS   int64   `json:"tls"`
}

type CustomUI struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Count int64  `json:"count"`
	Value string `json:"value"`
}

type HTMLReportData struct {
	TestName        string         `json:"test_name"`
	StartTime       string         `json:"start_time"`
	DurationSeconds float64        `json:"duration_seconds"`
	TotalRequests   int64          `json:"total_requests"`
	Success         int64          `json:"success"`
	NetworkFails    int64          `json:"network_fails"`
	LogicFails      int64          `json:"logic_fails"`
	AverageRPS      float64        `json:"average_rps"`
	BytesSent       int64          `json:"bytes_sent"`
	BytesRecv       int64          `json:"bytes_recv"`
	IterAvg         int64          `json:"iter_avg"`
	IterP95         int64          `json:"iter_p95"`
	GlobalReqP95    int64          `json:"global_req_p95"`
	GlobalReqP99    int64          `json:"global_req_p99"`
	TimeSeries      []TimeSeriesUI `json:"time_series"`
	Nodes           []NodeUI       `json:"nodes"`
	StatusCodes     map[string]int `json:"status_codes"`
	CustomMetrics   []CustomUI     `json:"custom_metrics"`
	Flows           []FlowUI       `json:"flows"`
	TopErrors       map[string]int `json:"top_errors"`
	ThresholdPassed bool           `json:"threshold_passed"`
	Thresholds      []ThresholdUI  `json:"thresholds"`
	ASTTree         string         `json:"ast_tree"`
}

type HTMLReporter struct {
	StartTime time.Time
	FilePath  string
	fs        filesystem.FS
	Scenarios []ExportScenario
}

func NewHTMLReporter(filepath string, fs filesystem.FS, startTime time.Time, scenarios []ExportScenario) *HTMLReporter {
	if filepath == "" {
		filepath = "blaster-report.html"
	}
	return &HTMLReporter{StartTime: startTime, FilePath: filepath, fs: fs, Scenarios: scenarios}
}

func (r *HTMLReporter) Export(engine *metrics.EngineMetrics, assert *metrics.AssertionManager) error {
	total, _, success, netFail, logicFail, _, dataSent, dataRecv := engine.Snapshot()
	durSecs := time.Since(r.StartTime).Seconds()
	if durSecs <= 0 {
		durSecs = 0.001
	}

	var rps float64
	if total > 0 {
		rps = float64(total) / durSecs
	}

	var thresholdList []ThresholdUI
	thresholdPassed := true
	if assert != nil {
		_ = assert.EvaluateThresholds(engine)
		thresholdPassed = !assert.HasFailures()
		for _, res := range assert.GetResults() {
			thresholdList = append(thresholdList, ThresholdUI{
				Metric:   res.Metric,
				Criteria: res.Criteria,
				Actual:   res.Actual,
				Passed:   res.Passed,
				Reason:   res.Reason,
			})
		}
	}

	var globalP95, globalP99 int64
	if engine.RequestDuration != nil && total > 0 {
		globalP95 = engine.RequestDuration.Percentile(95.0)
		globalP99 = engine.RequestDuration.Percentile(99.0)
	}

	data := HTMLReportData{
		TestName:        "Blaster Load Test Execution",
		StartTime:       r.StartTime.Format(time.RFC1123),
		DurationSeconds: durSecs,
		TotalRequests:   total,
		Success:         success,
		NetworkFails:    netFail,
		LogicFails:      logicFail,
		AverageRPS:      rps,
		BytesSent:       dataSent,
		BytesRecv:       dataRecv,
		IterAvg:         engine.IterationDuration.Percentile(50.0),
		IterP95:         engine.IterationDuration.Percentile(95.0),
		GlobalReqP95:    globalP95,
		GlobalReqP99:    globalP99,
		StatusCodes:     make(map[string]int),
		TopErrors:       engine.GetTopErrors(),
		ThresholdPassed: thresholdPassed,
		Thresholds:      thresholdList,
		ASTTree:         generateTextTree(r.Scenarios),
		TimeSeries:      make([]TimeSeriesUI, 0, len(engine.TimeSeries)),
		Nodes:           make([]NodeUI, 0),
		CustomMetrics:   make([]CustomUI, 0),
		Flows:           make([]FlowUI, 0),
	}

	for _, pt := range engine.TimeSeries {
		timeStr := time.Unix(pt.Timestamp, 0).Format("15:04:05")
		data.TimeSeries = append(data.TimeSeries, TimeSeriesUI{
			Time: timeStr,
			RPS:  pt.RPS,
			VUs:  pt.VUs,
		})
	}

	// Endpoints: Sorted in descending order by request count
	nodeMetrics := engine.GetAllNodes()
	for id, nodeMet := range nodeMetrics {
		reqs := atomic.LoadInt64(&nodeMet.TotalRequests)
		if reqs == 0 {
			continue
		}
		fails := atomic.LoadInt64(&nodeMet.FailCount)
		totalLat := atomic.LoadInt64(&nodeMet.TotalLatencyUs)

		parsedPath := extractPath(nodeMet.Name)
		displayName := id
		if nodeMet.Method != "" && parsedPath != "" {
			displayName = fmt.Sprintf("%-6s %s [%s]", nodeMet.Method, parsedPath, id)
		} else if parsedPath != "" {
			displayName = fmt.Sprintf("%s [%s]", parsedPath, id)
		}

		data.Nodes = append(data.Nodes, NodeUI{
			ID:    displayName,
			Reqs:  reqs,
			Fails: fails,
			Min:   nodeMet.ReqDuration.Min(),
			Avg:   (float64(totalLat) / float64(reqs)) / 1000.0,
			P90:   nodeMet.ReqDuration.Percentile(90.0),
			P95:   nodeMet.ReqDuration.Percentile(95.0),
			P99:   nodeMet.ReqDuration.Percentile(99.0),
			Max:   nodeMet.ReqDuration.Max(),
			TTFB:  nodeMet.TTFB.Percentile(90.0),
			TCP:   nodeMet.TCPConn.Percentile(90.0),
			TLS:   nodeMet.TLSHand.Percentile(90.0),
		})

		for code, count := range nodeMet.StatusCodes {
			if count > 0 {
				data.StatusCodes[fmt.Sprint(code)] += int(count)
			}
		}
	}
	sort.Slice(data.Nodes, func(i, j int) bool {
		if data.Nodes[i].Reqs == data.Nodes[j].Reqs {
			return data.Nodes[i].ID < data.Nodes[j].ID
		}
		return data.Nodes[i].Reqs > data.Nodes[j].Reqs
	})

	// Custom Metrics
	for k, c := range engine.Custom.GetAll() {
		valStr := fmt.Sprintf("%.2f", c.Last)
		switch c.Type {
		case metrics.TypeTrend:
			valStr = fmt.Sprintf("Last: %.2f | Min: %.2f | Max: %.2f", c.Last, c.Min, c.Max)
		case metrics.TypeCounter:
			valStr = fmt.Sprintf("Sum: %.2f", c.Sum)
		case metrics.TypeGauge:
			valStr = fmt.Sprintf("Current: %.2f", c.Last)
		}
		data.CustomMetrics = append(data.CustomMetrics, CustomUI{
			Name:  k,
			Type:  string(c.Type),
			Count: c.Count,
			Value: valStr,
		})
	}
	sort.Slice(data.CustomMetrics, func(i, j int) bool {
		return data.CustomMetrics[i].Name < data.CustomMetrics[j].Name
	})

	// Control Flow Analytics (Scripts, Branches, Loops, Polls, Matches)
	for id, s := range engine.GetAllScripts() {
		calls := atomic.LoadInt64(&s.TotalCalls)
		fails := atomic.LoadInt64(&s.FailCount)
		lat := atomic.LoadInt64(&s.TotalLatencyUs)
		avgMs := 0.0
		if calls > 0 {
			avgMs = (float64(lat) / float64(calls)) / 1000.0
		}
		data.Flows = append(data.Flows, FlowUI{ID: id, Type: "SCRIPT", Details: fmt.Sprintf("Calls: %d | Fails: %d | Avg: %.1f ms", calls, fails, avgMs)})
	}
	for id, b := range engine.GetAllBranches() {
		trueC := atomic.LoadInt64(&b.TrueCount)
		falseC := atomic.LoadInt64(&b.FalseCount)
		data.Flows = append(data.Flows, FlowUI{ID: id, Type: "BRANCH", Details: fmt.Sprintf("True: %d | False: %d", trueC, falseC)})
	}
	for id, l := range engine.GetAllLoops() {
		entered := atomic.LoadInt64(&l.TotalEntered)
		iters := atomic.LoadInt64(&l.TotalIterations)
		data.Flows = append(data.Flows, FlowUI{ID: id, Type: "LOOP", Details: fmt.Sprintf("Entered: %d times | Total Iterations: %d", entered, iters)})
	}
	for id, p := range engine.GetAllPolls() {
		entered := atomic.LoadInt64(&p.TotalEntered)
		successCount := atomic.LoadInt64(&p.SuccessCount)
		exhausted := atomic.LoadInt64(&p.Exhausted)
		data.Flows = append(data.Flows, FlowUI{ID: id, Type: "POLL", Details: fmt.Sprintf("Total: %d | Success: %d | Exhausted: %d", entered, successCount, exhausted)})
	}
	for id, m := range engine.GetAllMatches() {
		cases := m.GetAll()
		var caseStrs []string
		for k, v := range cases {
			caseStrs = append(caseStrs, fmt.Sprintf("%s: %d", k, v))
		}
		data.Flows = append(data.Flows, FlowUI{ID: id, Type: "MATCH", Details: strings.Join(caseStrs, " | ")})
	}
	sort.Slice(data.Flows, func(i, j int) bool {
		return data.Flows[i].ID < data.Flows[j].ID
	})

	rawJSON, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal html report data: %w", err)
	}

	safeJSON := bytes.ReplaceAll(rawJSON, []byte("</"), []byte(`<\/`))

	tmpl, err := template.New("report").Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse html template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, string(safeJSON)); err != nil {
		return fmt.Errorf("failed to execute html template: %w", err)
	}

	if err := r.fs.Write(r.FilePath, buf.Bytes()); err != nil {
		return fmt.Errorf("failed to write HTML report to %s: %w", r.FilePath, err)
	}

	fmt.Printf("📊 [HTML Exporter] Report written to: %s\n", r.FilePath)
	return nil
}

func generateTextTree(scenarios []ExportScenario) string {
	var sb strings.Builder
	for i, scn := range scenarios {
		name := scn.Name
		if name == "" {
			name = fmt.Sprintf("Scenario %d", i+1)
		}
		sb.WriteString(fmt.Sprintf("▶ %s\n", name))

		if len(scn.Graph.Setup) > 0 {
			sb.WriteString("\n  [SETUP PHASE]\n")
			buildTextTree(scn.Graph.Setup, "    ", &sb)
		}

		if len(scn.Graph.Execution) > 0 {
			sb.WriteString("\n  [EXECUTION PHASE]\n")
			buildTextTree(scn.Graph.Execution, "    ", &sb)
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func summarizeNodesText(nodes []planner.ExecutableNode) string {
	if len(nodes) == 0 {
		return ""
	}
	var parts []string
	for _, n := range nodes {
		parts = append(parts, strings.ToLower(n.Type()))
	}
	return strings.Join(parts, ", ")
}

func buildTextTree(nodes []planner.ExecutableNode, prefix string, sb *strings.Builder) {
	for i, node := range nodes {
		isLast := i == len(nodes)-1
		connector := "├──"
		childPrefix := prefix + "│   "
		if isLast {
			connector = "└──"
			childPrefix = prefix + "    "
		}

		nodeID := fmt.Sprintf("[%s]", node.NodeID())

		switch n := node.(type) {
		case *planner.ActionNode:
			fmt.Fprintf(sb, "%s%s [%s] %s %s %s\n", prefix, connector, n.Protocol, n.Method, html.EscapeString(n.Target.Raw), nodeID)
			if len(n.BeforePipeline) > 0 {
				fmt.Fprintf(sb, "%s├── before → %s\n", childPrefix, summarizeNodesText(n.BeforePipeline))
			}
			if len(n.AfterPipeline) > 0 {
				fmt.Fprintf(sb, "%s└── after → %s\n", childPrefix, summarizeNodesText(n.AfterPipeline))
			}
		case *planner.BranchNode:
			sb.WriteString(fmt.Sprintf("%s%s [BRANCH] %s\n", prefix, connector, nodeID))
			sb.WriteString(fmt.Sprintf("%s├── [TRUE]\n", childPrefix))
			buildTextTree(n.TruePath, childPrefix+"│   ", sb)
			if len(n.FalsePath) > 0 {
				sb.WriteString(fmt.Sprintf("%s└── [FALSE]\n", childPrefix))
				buildTextTree(n.FalsePath, childPrefix+"    ", sb)
			} else {
				sb.WriteString(fmt.Sprintf("%s└── [FALSE] (empty)\n", childPrefix))
			}
		case *planner.LoopNode:
			sb.WriteString(fmt.Sprintf("%s%s [LOOP] %d iterations %s\n", prefix, connector, n.Count, nodeID))
			buildTextTree(n.Logic, childPrefix, sb)
		case *planner.MatchNode:
			sb.WriteString(fmt.Sprintf("%s%s [MATCH] %s\n", prefix, connector, nodeID))
			for caseName, path := range n.Cases {
				if len(path) > 0 {
					sb.WriteString(fmt.Sprintf("%s├── Case: %s\n", childPrefix, caseName))
					buildTextTree(path, childPrefix+"│   ", sb)
				} else {
					sb.WriteString(fmt.Sprintf("%s├── Case: %s (empty)\n", childPrefix, caseName))
				}
			}
			if len(n.DefaultPath) > 0 {
				sb.WriteString(fmt.Sprintf("%s└── Default\n", childPrefix))
				buildTextTree(n.DefaultPath, childPrefix+"    ", sb)
			}
		case *planner.PollNode:
			sb.WriteString(fmt.Sprintf("%s%s [POLL] Max: %d %s\n", prefix, connector, n.MaxAttempts, nodeID))
			sb.WriteString(fmt.Sprintf("%s└── [LOGIC]\n", childPrefix))
			buildTextTree(n.Logic, childPrefix+"    ", sb)
		default:
			sb.WriteString(fmt.Sprintf("%s%s [%s] %s\n", prefix, connector, node.Type(), nodeID))
		}
	}
}
