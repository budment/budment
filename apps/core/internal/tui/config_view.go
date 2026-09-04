package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/vunas/blaster/internal/config"
	"github.com/vunas/blaster/internal/tui/theme"
	"github.com/vunas/blaster/internal/tui/widgets"
)

func PrintPlanOverview(scriptPath string, compileTimeMs int64, cfg config.EngineConfig) {
	table := widgets.NewTable("Configuration", "Resolved Value")

	if scriptPath != "" {
		table.AddRow("Target Script", scriptPath)
	}
	if compileTimeMs > 0 {
		table.AddRow("Compile Time", fmt.Sprintf("%d ms", compileTimeMs))
	}

	if len(cfg.Stages) > 0 {
		var totalDur time.Duration
		peakVUs := 0
		for _, stg := range cfg.Stages {
			d, _ := time.ParseDuration(stg.Duration)
			totalDur += d
			if stg.Target > peakVUs {
				peakVUs = stg.Target
			}
		}
		table.AddRow("Execution Mode", fmt.Sprintf("Ramping Stages (%d stages)", len(cfg.Stages)))
		table.AddRow("Peak Virtual Users", fmt.Sprintf("%d VUs", peakVUs))
		table.AddRow("Planned Duration", totalDur.String())
	} else {
		table.AddRow("Execution Mode", "Constant VUs")
		table.AddRow("Virtual Users", fmt.Sprintf("%d VUs", cfg.VUs))
		durText := cfg.Duration
		if durText == "" {
			durText = "until iterations complete"
		}
		table.AddRow("Target Duration", durText)
	}

	if cfg.MaxDuration != "" && cfg.MaxDuration != "0s" {
		table.AddRow("Max Duration (Limit)", cfg.MaxDuration)
	}
	if cfg.Iterations > 0 {
		table.AddRow("Total Iterations", fmt.Sprintf("%d iters", cfg.Iterations))
	}

	if cfg.InsecureSkipTLS {
		table.AddRow("TLS Verification", theme.TextYellow("Insecure / Skipped"))
	} else {
		table.AddRow("TLS Verification", "Strict")
	}

	if len(cfg.Thresholds) > 0 {
		var ths []string
		for k, v := range cfg.Thresholds {
			ths = append(ths, fmt.Sprintf("%s (%s)", k, v))
		}
		table.AddRow("Quality Gates (SLA)", strings.Join(ths, ", "))
	}

	if len(cfg.Tags) > 0 {
		var tags []string
		for k, v := range cfg.Tags {
			tags = append(tags, fmt.Sprintf("%s=%s", k, v))
		}
		table.AddRow("Metadata Tags", strings.Join(tags, ", "))
	}

	fmt.Print(table.Render())
}

func PrintDivider() {
	fmt.Println(theme.TextDim(strings.Repeat("-", 80)))
}
