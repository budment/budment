package cli

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/vunas/blaster/internal/config"
	"github.com/vunas/blaster/internal/filesystem"
	"github.com/vunas/blaster/internal/pipeline"
	"github.com/vunas/blaster/internal/tui"
	"github.com/vunas/blaster/internal/tui/theme"
)

var planDetailed bool

var planCmd = &cobra.Command{
	Use:   "plan [script.ts]",
	Short: "Build and inspect the static execution plan",
	Args:  cobra.ExactArgs(1),

	RunE: func(cmd *cobra.Command, args []string) error {
		scriptPath := args[0]
		start := time.Now()

		fs := filesystem.NewLocal()
		loader := config.NewLoader(globalConfigFile, fs)
		projectCfg, _ := loader.Load()
		planResult, err := pipeline.BuildPlan(scriptPath)
		if err != nil {
			if !quietMode {
				fmt.Printf("\n%s %s\n", theme.TextRed("✗"), theme.TextRed("Plan generation failed"))
				fmt.Printf("  %s\n", err)
			}
			return err
		}

		cliOverrides := config.CLIConfig{}
		if cmd.Flags().Changed("vus") {
			cliOverrides.VUs = &cliVUs
		}
		if cmd.Flags().Changed("duration") {
			cliOverrides.Duration = &cliDuration
		}
		envOverrides := config.ParseEnv()
		finalCfg := config.MergeEngineConfig(projectCfg, planResult.ASTConfig, envOverrides, cliOverrides)
		elapsed := time.Since(start)

		if quietMode {
			return nil
		}

		if jsonMode {
			output := map[string]any{
				"target":          scriptPath,
				"compile_time_ms": elapsed.Milliseconds(),
				"resolved_config": finalCfg,
				"scenarios":       planResult.Scenarios,
			}
			rawJSON, _ := json.MarshalIndent(output, "", "  ")
			fmt.Println(string(rawJSON))
			return nil
		}

		fmt.Printf("%s %s\n", theme.TextCyan("BLASTER PLANNER"), theme.TextDim("· STATIC PLAN"))
		fmt.Println(theme.TextDim("────────────────────────────────────────────────────────────"))
		fmt.Printf("  Target          : %s\n", theme.TextBold(scriptPath))
		fmt.Printf("  Compile Time    : %d ms\n", elapsed.Milliseconds())
		fmt.Println(theme.TextDim("────────────────────────────────────────────────────────────"))

		if debugMode {
			fmt.Printf("\n%s\n", theme.TextCyan("EXECUTION PLAN · AST GRAPH"))
			fmt.Println(theme.TextDim("────────────────────────────────────────────────────────────"))
			fmt.Println(tui.HighlightJSON(string(planResult.RawASTJSON)))
			fmt.Println(theme.TextDim("────────────────────────────────────────────────────────────"))
		}
		fmt.Printf("\n%s\n", theme.TextCyan("EXECUTION PLAN · LIFECYCLE TREE"))
		fmt.Println(theme.TextDim("────────────────────────────────────────────────────────────"))

		isDetailed := planDetailed || debugMode
		// Render the complete lifecycle tree for every scenario.
		for i, scn := range planResult.Scenarios {
			name := scn.Name
			if name == "" {
				name = fmt.Sprintf("Scenario %d", i+1)
			}

			fmt.Printf("\n%s %s\n", theme.TextMagenta("▶"), theme.TextBold(name))
			tui.PrintPhase("setup", scn.Graph.Setup, isDetailed)
			tui.PrintPhase("execution", scn.Graph.Execution, isDetailed)
		}
		fmt.Println("\n" + theme.TextDim("────────────────────────────────────────────────────────────"))
		fmt.Printf("%s Plan is valid. Run %s to execute.\n", theme.TextGreen("✓"), theme.TextBold(fmt.Sprintf("blaster run %s", scriptPath)))

		return nil
	},
}

func init() {
	rootCmd.AddCommand(planCmd)
	planCmd.Flags().IntVarP(&cliVUs, "vus", "v", 1, "Override concurrent Virtual Users")
	planCmd.Flags().StringVarP(&cliDuration, "duration", "d", "0s", "Override test duration (e.g., 30s, 5m)")
	planCmd.Flags().BoolVar(&planDetailed, "detail", false, "Print detailed execution tree with node IDs")
}
