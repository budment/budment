package tui

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/budment/budment/internal/config"
	"github.com/budment/budment/internal/tui/theme"
)

func visibleLen(s string) int {
	return utf8.RuneCountInString(ansiRegexp.ReplaceAllString(s, ""))
}

type propItem struct {
	key  string
	val  string
	wide bool
}

func PrintPlanOverview(scriptPath string, compileTimeMs int64, cfg config.EngineConfig) {
	var compact []propItem
	var wide []propItem

	if scriptPath != "" {
		compact = append(compact, propItem{key: "Target Script", val: scriptPath})
	}
	if compileTimeMs > 0 {
		compact = append(compact, propItem{key: "Compile Time", val: fmt.Sprintf("%d ms", compileTimeMs)})
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
		compact = append(compact,
			propItem{key: "Execution Mode", val: fmt.Sprintf("Ramping Stages (%d stages)", len(cfg.Stages))},
			propItem{key: "Peak Virtual Users", val: fmt.Sprintf("%d VUs", peakVUs)},
			propItem{key: "Planned Duration", val: totalDur.String()},
		)
	} else {
		durText := cfg.Duration
		if durText == "" {
			durText = "until iterations complete"
		}
		compact = append(compact,
			propItem{key: "Execution Mode", val: "Constant VUs"},
			propItem{key: "Virtual Users", val: fmt.Sprintf("%d VUs", cfg.VUs)},
			propItem{key: "Target Duration", val: durText},
		)
	}

	if cfg.MaxDuration != "" && cfg.MaxDuration != "0s" {
		compact = append(compact, propItem{key: "Max Duration", val: cfg.MaxDuration})
	}
	if cfg.Iterations != nil {
		compact = append(compact, propItem{key: "Total Iterations", val: fmt.Sprintf("%d iters", cfg.Iterations)})
	}

	if cfg.InsecureSkipTLS {
		compact = append(compact, propItem{key: "TLS Verification", val: theme.TextYellow("Insecure / Skipped")})
	} else {
		compact = append(compact, propItem{key: "TLS Verification", val: "Strict"})
	}

	if len(cfg.Thresholds) > 0 {
		ths := make([]string, 0, len(cfg.Thresholds))
		for k, v := range cfg.Thresholds {
			ths = append(ths, fmt.Sprintf("%s (%s)", k, v))
		}
		wide = append(wide, propItem{key: "Quality Gates", val: strings.Join(ths, ", "), wide: true})
	}

	if len(cfg.Tags) > 0 {
		tags := make([]string, 0, len(cfg.Tags))
		for k, v := range cfg.Tags {
			tags = append(tags, fmt.Sprintf("%s=%s", k, v))
		}
		wide = append(wide, propItem{key: "Metadata Tags", val: strings.Join(tags, ", "), wide: true})
	}

	fmt.Printf("\n%s\n", theme.TextCyan("CONFIGURATION"))
	PrintDivider()
	col1KeyWidth := 0
	col1ValWidth := 0
	col2KeyWidth := 0

	for i := 0; i < len(compact); i += 2 {
		leftKey := compact[i].key + ":"
		if len(leftKey) > col1KeyWidth {
			col1KeyWidth = len(leftKey)
		}
		if v := visibleLen(compact[i].val); v > col1ValWidth {
			col1ValWidth = v
		}

		if i+1 < len(compact) {
			rightKey := compact[i+1].key + ":"
			if len(rightKey) > col2KeyWidth {
				col2KeyWidth = len(rightKey)
			}
		}
	}

	if col1ValWidth < 18 {
		col1ValWidth = 18
	}

	for i := 0; i < len(compact); i += 2 {
		left := compact[i]
		leftLabel := padRight(theme.TextDim(left.key+":"), col1KeyWidth+1)
		leftVal := padRight(left.val, col1ValWidth)

		if i+1 < len(compact) {
			right := compact[i+1]
			rightLabel := padRight(theme.TextDim(right.key+":"), col2KeyWidth+1)
			fmt.Printf("  %s %s    %s %s\n", leftLabel, leftVal, rightLabel, right.val)
		} else {
			fmt.Printf("  %s %s\n", leftLabel, leftVal)
		}
	}

	for _, item := range wide {
		label := padRight(theme.TextDim(item.key+":"), col1KeyWidth+1)
		fmt.Printf("  %s %s\n", label, item.val)
	}

	PrintDivider()
}

func PrintDivider() {
	fmt.Println(theme.TextDim(strings.Repeat("-", 80)))
}
