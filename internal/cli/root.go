package cli

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/vunas/blaster/internal/logger"
)

var (
	globalLog *slog.Logger

	globalConfigFile string
	debugMode        bool
	quietMode        bool
	jsonMode         bool
)

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "blaster",
	Short: "Blaster - OpenAPI-driven API testing engine",
	Long: `Blaster is a deterministic API testing engine that generates
workflows and execution plans directly from OpenAPI specifications.`,

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
		if globalLog != nil {
			globalLog.Error("Execution failed", "err", err)
		} else {
			fmt.Fprintln(os.Stderr, err)
		}
		return 1
	}
	return 0
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&globalConfigFile, "config", "c", "blaster.yaml", "Path to global project config file")
	rootCmd.PersistentFlags().BoolVar(&debugMode, "debug", false, "Enable verbose debug logging")
	rootCmd.PersistentFlags().BoolVar(&quietMode, "quiet", false, "Suppress all logs")
	rootCmd.PersistentFlags().BoolVar(&jsonMode, "json", false, "Output logs in JSON format")
}
