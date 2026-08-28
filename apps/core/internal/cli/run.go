package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/vunas/blaster/internal/config"
	"github.com/vunas/blaster/internal/exporters"
	"github.com/vunas/blaster/internal/filesystem"
	"github.com/vunas/blaster/internal/metrics"
	"github.com/vunas/blaster/internal/pipeline"
	"github.com/vunas/blaster/internal/tui"
	"github.com/vunas/blaster/internal/tui/theme"
)

var (
	cliVUs         int
	cliDuration    string
	endpointDirRun string
)

type cliMetricsSink struct {
	agg       *metrics.Aggregator
	dashboard *tui.LiveDashboard
}

func (c *cliMetricsSink) Log(workerID int, nodeID string, level string, msg string) {
	timeStr := time.Now().Format("15:04:05.000")

	var coloredLog string

	if level == "SYS_ERR" && c.dashboard != nil {
		return
	}

	hookCtx := ""
	if nodeID != "unknown" && nodeID != "" {
		hookCtx = fmt.Sprintf("[Hook: %s] ", nodeID)
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
		coloredLog = fmt.Sprintf("%s %s | %s %s[System] %s", prefix, theme.TextRed("ERROR"), msg, vuInfo, hookCtx)
	}

	if c.dashboard != nil {
		select {
		case c.dashboard.LogChan <- coloredLog:
		default:
		}
	} else if globalLog != nil {
		switch level {
		case "WARN":
			globalLog.Warn(fmt.Sprintf("[VU:%d] %s%s", workerID, hookCtx, msg))

		case "ERROR", "SYS_ERR":
			globalLog.Error(fmt.Sprintf("[VU:%d] %s%s", workerID, hookCtx, msg))

		default:
			globalLog.Info(fmt.Sprintf("[VU:%d] %s%s", workerID, hookCtx, msg))
		}
	}
}

func (c *cliMetricsSink) RecordEvent(workerID int, nodeID string, eventType string, reason string,
) {
	if eventType == "FAIL" || eventType == "ABORT" {
		errMsg := fmt.Sprintf("[%s] %s", eventType, reason)

		if c.agg != nil {
			c.agg.PushEvent(metrics.MetricEvent{WorkerID: workerID, NodeID: nodeID, ErrorMsg: errMsg, IsLogicFail: true})
		}

		c.Log(workerID, nodeID, "ERROR", errMsg)
	}
}

func (c *cliMetricsSink) Tag(workerID int, key string, value string) {
	if globalLog != nil {
		globalLog.Debug(
			fmt.Sprintf("[VU:%d] Tag: %s = %s", workerID, key, value),
		)
	}
}

func (c *cliMetricsSink) RecordCustom(workerID int, mType string, name string, val float64,
) {
	if c.agg != nil {
		c.agg.PushEvent(metrics.MetricEvent{WorkerID: workerID, IsCustom: true, CustomType: metrics.CustomMetricType(mType), CustomName: name, CustomVal: val})
	}
}

var runCmd = &cobra.Command{
	Use:   "run [script.ts]",
	Short: "Execute a load testing scenario from a TypeScript/JavaScript file",
	Args:  cobra.ExactArgs(1),

	RunE: func(cmd *cobra.Command, args []string) error {
		defer tui.ShowCursor()
		scriptPath := args[0]
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
		tempCfg := config.MergeEngineConfig(projectCfg, config.ASTConfig{}, envOverrides, cliOverrides)
		isTUIActive := !quietMode && !jsonMode && !debugMode

		if !isTUIActive {
			globalLog.Info("Starting Blaster execution phase", "script", scriptPath)
			globalLog.Info("Preparing engine and virtual machines")
		}

		planResult, err := pipeline.BuildPlan(scriptPath, endpointDirRun, tempCfg.AutoPlumb, fs)
		if err != nil {
			return fmt.Errorf("build plan failed: %w", err)
		}

		// Debug output: compiled execution graph and lifecycle tree.
		if debugMode {
			fmt.Printf("\n%s\n", theme.TextCyan("EXECUTION PLAN · AST GRAPH"))
			fmt.Println(theme.TextDim("────────────────────────────────────────────────────────────"))
			fmt.Println(tui.HighlightJSON(string(planResult.RawASTJSON)))
			fmt.Printf("\n%s\n", theme.TextCyan("EXECUTION PLAN · LIFECYCLE TREE"))
			fmt.Println(theme.TextDim("────────────────────────────────────────────────────────────"))

			for i, scn := range planResult.Scenarios {
				name := scn.Name
				if name == "" {
					name = fmt.Sprintf("Scenario %d", i+1)
				}
				fmt.Printf("\n%s %s\n", theme.TextMagenta("▶"), theme.TextBold(name))
				tui.PrintPhase("setup", scn.Graph.Setup)
				tui.PrintPhase("execution", scn.Graph.Execution)
			}
		}
		finalEngineCfg := config.MergeEngineConfig(projectCfg, planResult.ASTConfig, envOverrides, cliOverrides)
		testStartTime := time.Now()

		var reporters []exporters.Reporter
		reporters = append(reporters, exporters.NewTerminalReporter(testStartTime))
		reporters = append(
			reporters,
			exporters.NewHTMLReporter(finalEngineCfg.Exporters.HTML, fs, testStartTime),
		)
		agg := metrics.NewAggregator(1000000)
		sink := &cliMetricsSink{
			agg: agg,
		}
		app, err := pipeline.PrepareExecution(planResult, finalEngineCfg, globalLog, agg, sink, reporters)
		if err != nil {
			return fmt.Errorf("pipeline preparation failed: %w", err)
		}

		if isTUIActive {
			tui.PrintBanner()

			// Aggregate dashboard targets across all scenarios.
			var maxTargetVUs int
			var maxDuration time.Duration
			var totalIters int

			for _, scn := range app.Director.Scenarios {
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

				// Scenarios are assumed to run concurrently.
				maxTargetVUs += scnVUs
				if scnDur > maxDuration {
					maxDuration = scnDur
				}
				totalIters += scn.Config.Iterations
			}

			dashboard := tui.NewLiveDashboard(app.Aggregator.Metrics, maxTargetVUs, maxDuration, totalIters)
			sink.dashboard = dashboard
			dashCtx, dashCancel := context.WithCancel(
				context.Background(),
			)
			go dashboard.Start(dashCtx)
			defer dashCancel()

		} else {
			globalLog.Info("Deploying engine director")
		}

		return app.Director.Run()
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().IntVarP(&cliVUs, "vus", "v", 1, "Number of concurrent Virtual Users")
	runCmd.Flags().StringVarP(&cliDuration, "duration", "d", "0s", "Test duration (e.g., 30s, 5m)")
	runCmd.Flags().StringVarP(&endpointDirRun, "endpoint", "e", "endpoint", "Path to endpoint definitions directory")
}
