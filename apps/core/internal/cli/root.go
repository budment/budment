package cli

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/vunas/blaster/internal/logger"
)

var (
	globalLog *slog.Logger

	globalConfigFile string
	debugMode        bool
	quietMode        bool
	jsonMode         bool
	noTUIMode        bool
)

func IsTerminal() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

func IsCIEnvironment() bool {
	return os.Getenv("CI") != "" || os.Getenv("CONTINUOUS_INTEGRATION") != "" || os.Getenv("BUILD_NUMBER") != ""
}

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:           "blaster",
	Short:         "Blaster - Testing Engine",
	SilenceErrors: true,
	SilenceUsage:  true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		globalLog = logger.New(logger.Config{
			Verbose: debugMode,
			Quiet:   quietMode,
			JSON:    jsonMode,
		})
		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() int {
	if err := rootCmd.Execute(); err != nil {
		if !quietMode {
			if globalLog != nil {
				globalLog.Error("Execution failed", "err", err)
			} else {
				fmt.Fprintln(os.Stderr, "Error:", err)
			}
		}
		return 1
	}
	return 0
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&globalConfigFile, "config", "c", "blaster.yaml", "Path to global project config file")
	rootCmd.PersistentFlags().BoolVar(&debugMode, "debug", false, "Enable verbose debug logging")
	rootCmd.PersistentFlags().BoolVarP(&quietMode, "quiet", "q", false, "Suppress all terminal output (returns only exit codes)")
	rootCmd.PersistentFlags().BoolVar(&jsonMode, "json", false, "Output logs and reports in JSON format")
	rootCmd.PersistentFlags().BoolVar(&noTUIMode, "no-tui", false, "Disable interactive TUI dashboard (auto-enabled in CI/Non-TTY)")
}
