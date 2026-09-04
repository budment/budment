package cli

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/spf13/cobra"

	"github.com/vunas/blaster/internal/config"
	"github.com/vunas/blaster/internal/engine"
	"github.com/vunas/blaster/internal/exporters"
	"github.com/vunas/blaster/internal/filesystem"
	"github.com/vunas/blaster/internal/metrics"
	"github.com/vunas/blaster/internal/pipeline"
	"github.com/vunas/blaster/internal/tui"
	"github.com/vunas/blaster/internal/tui/theme"
)

var (
	cliVUs      int
	cliDuration string
	runDetailed bool
)

type cliMetricsSink struct {
	agg       *metrics.Aggregator
	dashboard *tui.LiveDashboard
}

func (c *cliMetricsSink) Log(workerID int, nodeID string, level string, msg string) {
	if quietMode {
		return
	}

	timeStr := time.Now().Format("15:04:05.000")

	var coloredLog string

	if level == "SYS_ERR" && c.dashboard != nil {
		return
	}

	hookCtx := ""
	if nodeID != "unknown" && nodeID != "" {
		hookCtx = fmt.Sprintf("[Node: %s] ", nodeID)
	}

	prefix := theme.TextDim(fmt.Sprintf("[%s]", timeStr))
	vuInfo := fmt.Sprintf("[VU:%d]", workerID)

	switch level {
	case "INFO":
		coloredLog = fmt.Sprintf("%s %s | %s %s%s", prefix, theme.TextBlue(level), vuInfo, hookCtx, msg)
	case "WARN":
		coloredLog = fmt.Sprintf("%s %s | %s %s%s", prefix, theme.TextYellow(level), vuInfo, hookCtx, msg)
	case "ERROR":
		coloredLog = fmt.Sprintf("%s %s | %s %s%s", prefix, theme.TextRed(level), vuInfo, hookCtx, msg)
	case "SYS_ERR":
		coloredLog = fmt.Sprintf("%s %s | %s %s[System] %s", prefix, theme.TextRed("SYS_ERR"), vuInfo, hookCtx, msg)
	}

	if c.dashboard != nil {
		select {
		case c.dashboard.LogChan <- coloredLog:
		default:
		}
		return
	}

	if globalLog != nil {
		switch level {
		case "WARN":
			globalLog.Warn(msg, "vu", workerID, "node", nodeID)
		case "ERROR":
			globalLog.Error(msg, "vu", workerID, "node", nodeID)
		case "SYS_ERR":
			if debugMode {
				globalLog.Error(msg, "vu", workerID, "node", nodeID)
			}
		default:
			globalLog.Info(msg, "vu", workerID, "node", nodeID)
		}
	}
}

func (c *cliMetricsSink) RecordEvent(workerID int, nodeID string, eventType string, reason string) {
	if eventType == "FAIL" || eventType == "ABORT" {
		errMsg := fmt.Sprintf("[%s] %s", eventType, reason)

		if c.agg != nil {
			c.agg.PushEvent(metrics.MetricEvent{
				WorkerID:    workerID,
				NodeID:      nodeID,
				ErrorMsg:    errMsg,
				IsLogicFail: true,
			})
		}
		c.Log(workerID, nodeID, "ERROR", errMsg)
	}
}

func (c *cliMetricsSink) Tag(workerID int, key string, value string) {
	if globalLog != nil && !quietMode && debugMode {
		globalLog.Debug("VU Tag", "vu", workerID, "key", key, "val", value)
	}
}

func (c *cliMetricsSink) RecordCustom(workerID int, mType string, name string, val float64) {
	if c.agg != nil {
		c.agg.PushEvent(metrics.MetricEvent{
			WorkerID:   workerID,
			IsCustom:   true,
			CustomType: metrics.CustomMetricType(mType),
			CustomName: name,
			CustomVal:  val,
		})
	}
}

var runCmd = &cobra.Command{
	Use:   "run [script.ts]",
	Short: "Execute a load testing scenario from a TypeScript/JavaScript file",
	Args:  cobra.ExactArgs(1),

	RunE: func(cmd *cobra.Command, args []string) error {
		defer tui.ShowCursor()
		scriptPath := args[0]
		start := time.Now()
		fs := filesystem.NewLocal()
		loader := config.NewLoader(globalConfigFile, fs)
		projectCfg, err := loader.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		envOverrides := config.ParseEnv()
		cliOverrides := config.CLIConfig{}
		if cmd.Flags().Changed("vus") {
			cliOverrides.VUs = &cliVUs
		}
		if cmd.Flags().Changed("duration") {
			cliOverrides.Duration = &cliDuration
		}

		planResult, err := pipeline.BuildPlan(scriptPath)
		if err != nil {
			return fmt.Errorf("build plan failed: %w", err)
		}

		finalEngineCfg := config.MergeEngineConfig(projectCfg, planResult.ASTConfig, envOverrides, cliOverrides)
		elapsed := time.Since(start)
		testStartTime := time.Now()

		isTUIActive := !quietMode && !jsonMode && !debugMode && !noTUIMode && IsTerminal() && !IsCIEnvironment()
		if !quietMode && !jsonMode {
			tui.PrintBanner()
			tui.PrintPlanOverview(scriptPath, elapsed.Milliseconds(), finalEngineCfg)
		}
		if !isTUIActive && !quietMode && !jsonMode {

			if debugMode || runDetailed {
				fmt.Printf("\n%s\n", theme.TextCyan("EXECUTION PLAN · LIFECYCLE TREE"))
				tui.PrintDivider()

				for i, scn := range planResult.Scenarios {
					name := scn.Name
					if name == "" {
						name = fmt.Sprintf("Scenario %d", i+1)
					}

					fmt.Printf("\n%s %s\n", theme.TextMagenta("◆"), theme.TextBold(name))
					tui.PrintPhase("setup", scn.Graph.Setup, runDetailed)
					tui.PrintPhase("execution", scn.Graph.Execution, runDetailed)
				}
				fmt.Println()
				tui.PrintDivider()
			}
			fmt.Printf("\n%s %s\n\n", theme.TextGreen("◆"), theme.TextBold("Starting scenario execution..."))
		}

		var exportScenarios []exporters.ExportScenario
		for _, scn := range planResult.Scenarios {
			exportScenarios = append(exportScenarios, exporters.ExportScenario{
				Name:  scn.Name,
				Graph: scn.Graph,
			})
		}

		var reporters []exporters.Reporter
		if !quietMode {
			reporters = append(reporters, exporters.NewTerminalReporter(testStartTime))
		}
		if finalEngineCfg.Exporters.HTML != "" {
			// reporters = append(reporters, exporters.NewHTMLReporter(finalEngineCfg.Exporters.HTML, fs, testStartTime, exportScenarios))
		}

		agg := metrics.NewAggregator(1000000)
		sink := &cliMetricsSink{agg: agg}

		app, err := pipeline.PrepareExecution(planResult, finalEngineCfg, agg, sink, nil)
		if err != nil {
			return fmt.Errorf("pipeline preparation failed: %w", err)
		}

		var dashCancel context.CancelFunc

		if isTUIActive {
			orderGroups := make(map[int][]*engine.Scenario)
			var orders []int
			for _, scn := range app.Director.Scenarios {
				ord := scn.Config.Order
				if _, ok := orderGroups[ord]; !ok {
					orders = append(orders, ord)
				}
				orderGroups[ord] = append(orderGroups[ord], scn)
			}
			sort.Ints(orders)

			var maxPeakVUs int
			var totalDuration time.Duration
			var totalIters int

			for _, ord := range orders {
				groupScenarios := orderGroups[ord]
				groupVUs := 0
				var groupMaxDur time.Duration

				for _, scn := range groupScenarios {
					scnVUs := scn.Config.VUs
					scnDur, _ := time.ParseDuration(scn.Config.Duration)

					// Resolve maximum VUs and total duration from stages.
					if len(scn.Config.Stages) > 0 {
						var stageDur time.Duration
						stageVUs := 0
						for _, stg := range scn.Config.Stages {
							d, _ := time.ParseDuration(stg.Duration)
							stageDur += d
							if stg.Target > stageVUs {
								stageVUs = stg.Target
							}
						}
						scnDur = stageDur
						scnVUs = stageVUs
					}

					groupVUs += scnVUs
					if scnDur > groupMaxDur {
						groupMaxDur = scnDur
					}
					totalIters += scn.Config.Iterations
				}

				if groupVUs > maxPeakVUs {
					maxPeakVUs = groupVUs
				}
				totalDuration += groupMaxDur
			}

			dashboard := tui.NewLiveDashboard(app.Aggregator.Metrics, maxPeakVUs, totalDuration, totalIters)
			sink.dashboard = dashboard

			var dashCtx context.Context
			dashCtx, dashCancel = context.WithCancel(context.Background())
			go dashboard.Start(dashCtx)
		}

		runErr := app.Director.Run()

		if isTUIActive && dashCancel != nil {
			dashCancel()
			tui.ShowCursor()
			time.Sleep(50 * time.Millisecond)
		}

		assertMgr := metrics.NewAssertionManager(finalEngineCfg.Thresholds)
		_ = assertMgr.EvaluateThresholds(app.Aggregator.Metrics)

		for _, rep := range reporters {
			if expErr := rep.Export(app.Aggregator.Metrics, assertMgr); expErr != nil && !quietMode {
				fmt.Printf("Exporter Warning: %v\n", expErr)
			}
		}

		if runErr != nil {
			return runErr
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().IntVarP(&cliVUs, "vus", "v", 1, "Number of concurrent Virtual Users")
	runCmd.Flags().StringVarP(&cliDuration, "duration", "d", "0s", "Test duration (e.g., 30s, 5m)")
	runCmd.Flags().BoolVar(&runDetailed, "detail", false, "Print detailed execution tree with node IDs and full nested pipelines")
}
