package cli

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/budment/budment/internal/logger"
)

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
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
	Use:           "budment",
	Short:         "Budment - Testing Engine",
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

func SetVersionInfo(v, c, d string) {
	Version = v
	Commit = c
	Date = d

	rootCmd.Version = v
	rootCmd.SetVersionTemplate(fmt.Sprintf("budment version %s (commit: %s, built at: %s)\n", v, c, d))
	rootCmd.Flags().BoolP("version", "v", false, "Show version information")
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() int {
	if err := rootCmd.Execute(); err != nil {
		if !quietMode && debugMode && globalLog != nil {
			globalLog.Error("Execution failed", "err", err)
		}
		return 1
	}
	return 0
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&globalConfigFile, "config", "c", "budment.yaml", "Path to global project config file")
	rootCmd.PersistentFlags().BoolVar(&debugMode, "debug", false, "Enable verbose debug logging")
	rootCmd.PersistentFlags().BoolVarP(&quietMode, "quiet", "q", false, "Suppress all terminal output (returns only exit codes)")
	rootCmd.PersistentFlags().BoolVar(&jsonMode, "json", false, "Output logs and reports in JSON format")
	rootCmd.PersistentFlags().BoolVar(&noTUIMode, "no-tui", false, "Disable interactive TUI dashboard (auto-enabled in CI/Non-TTY)")
}
