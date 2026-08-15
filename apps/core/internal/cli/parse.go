package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/vunas/blaster/internal/config"
	"github.com/vunas/blaster/internal/filesystem"
	"github.com/vunas/blaster/internal/pipeline"
)

var (
	specFile    string
	endpointDir string
)

type BlasterYAML struct {
	AIConfig config.AIConfig `yaml:"ai"`
}

var parseCmd = &cobra.Command{
	Use:          "parse",
	Short:        "Analyzes API Specification and generates deterministic Endpoints",
	SilenceUsage: true,

	RunE: func(cmd *cobra.Command, args []string) error {

		// Load Blaster YAML Config
		var projectConfig BlasterYAML
		if data, err := os.ReadFile(globalConfigFile); err == nil {
			expandedData := []byte(os.ExpandEnv(string(data)))
			if err := yaml.Unmarshal(expandedData, &projectConfig); err != nil {
				return fmt.Errorf("invalid global config format in %s: %w", globalConfigFile, err)
			}
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("failed to read global config: %w", err)
		}

		// Load endpoint config
		endpointCfg := config.DefaultEndpointConfig()
		endpointCfgPath := filepath.Join(endpointDir, "endpoint.config.yaml")

		if data, err := os.ReadFile(endpointCfgPath); err == nil {
			if err := yaml.Unmarshal(data, &endpointCfg); err != nil {
				return fmt.Errorf("invalid endpoint config format in %s: %w", endpointCfgPath, err)
			}
			globalLog.Debug("Loaded custom endpoint config", "file", endpointCfgPath)
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("failed to read endpoint config: %w", err)
		} else {
			globalLog.Debug("Endpoint config not found, using defaults", "file", endpointCfgPath)
		}

		// Assemble domain config
		cfg := config.ParseConfig{
			SpecFile:       specFile,
			EndpointDir:    endpointDir,
			AIConfig:       projectConfig.AIConfig,
			EndpointConfig: endpointCfg,
		}

		fs := filesystem.NewLocal()

		return pipeline.RunParse(cfg, fs, globalLog)
	},
}

func init() {
	rootCmd.AddCommand(parseCmd)

	parseCmd.Flags().StringVarP(&specFile, "file", "f", "openapi.yaml", "Path to API specification file (OpenAPI, gRPC, etc.)")
	parseCmd.Flags().StringVarP(&endpointDir, "out", "o", "endpoint", "Output directory for endpoints")
}
