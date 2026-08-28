package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/vunas/blaster/internal/config"
	"github.com/vunas/blaster/internal/filesystem"
	"github.com/vunas/blaster/internal/pipeline"
	"github.com/vunas/blaster/internal/tui"
	"github.com/vunas/blaster/internal/tui/theme"
)

var endpointDirPlan string

var planCmd = &cobra.Command{
	Use:   "plan [script.ts]",
	Short: "Build and inspect the static execution plan",
	Args:  cobra.ExactArgs(1),

	RunE: func(cmd *cobra.Command, args []string) error {
		scriptPath := args[0]

		fmt.Printf("%s %s\n", theme.TextCyan("BLASTER PLANNER"), theme.TextDim("· STATIC PLAN"))
		fmt.Println(theme.TextDim("────────────────────────────────────────────────────────────"))
		fmt.Printf("  Target    %s\n", theme.TextBold(scriptPath))
		fmt.Println(theme.TextDim("────────────────────────────────────────────────────────────"))

		start := time.Now()

		fmt.Printf("\n%s %s\n", theme.TextCyan("▶"), "Building execution graph and resolving Auto-Plumbing...")

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

		finalEngineCfg := config.MergeEngineConfig(projectCfg, config.ASTConfig{}, envOverrides, cliOverrides)
		planResult, err := pipeline.BuildPlan(scriptPath, endpointDirPlan, finalEngineCfg.AutoPlumb, fs)
		if err != nil {
			fmt.Printf("\n%s %s\n", theme.TextRed("✗"), theme.TextRed("Plan generation failed"))
			fmt.Printf("  %s\n", err)
			return err
		}

		elapsed := time.Since(start)

		fmt.Printf("\n%s Plan generated successfully\n", theme.TextGreen("✓"))
		fmt.Printf("  Endpoints  %d\n", planResult.EndpointsCount)
		fmt.Printf("  Duration   %d ms\n", elapsed.Milliseconds())
		if debugMode {
			fmt.Printf("\n%s\n", theme.TextCyan("EXECUTION PLAN · AST GRAPH"))
			fmt.Println(theme.TextDim("────────────────────────────────────────────────────────────"))
			fmt.Println(tui.HighlightJSON(string(planResult.RawASTJSON)))
			fmt.Println(theme.TextDim("────────────────────────────────────────────────────────────"))
		}
		fmt.Printf("\n%s\n", theme.TextCyan("EXECUTION PLAN · LIFECYCLE TREE"))
		fmt.Println(theme.TextDim("────────────────────────────────────────────────────────────"))

		// Render the complete lifecycle tree for every scenario.
		for i, scn := range planResult.Scenarios {
			name := scn.Name
			if name == "" {
				name = fmt.Sprintf("Scenario %d", i+1)
			}

			fmt.Printf("\n%s %s\n", theme.TextMagenta("▶"), theme.TextBold(name))
			tui.PrintPhase("setup", scn.Graph.Setup)
			tui.PrintPhase("execution", scn.Graph.Execution)
		}
		fmt.Println("\n" + theme.TextDim("────────────────────────────────────────────────────────────"))
		fmt.Printf("%s Plan is valid. Run %s to execute.\n", theme.TextGreen("✓"), theme.TextBold(fmt.Sprintf("blaster run %s", scriptPath)))

		return nil
	},
}

func init() {
	rootCmd.AddCommand(planCmd)
	planCmd.Flags().StringVarP(&endpointDirPlan, "endpoint", "e", "endpoint", "Path to endpoint definitions directory")
	planCmd.Flags().IntVarP(&cliVUs, "vus", "v", 1, "Number of concurrent Virtual Users")
	planCmd.Flags().StringVarP(&cliDuration, "duration", "d", "0s", "Test duration (e.g., 30s, 5m)")
}
