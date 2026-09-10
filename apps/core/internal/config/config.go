package config

// Handles report exports to various formats
type ExportersConfig struct {
	JSON       string `yaml:"json"`
	HTML       string `yaml:"html"`
	Prometheus string `yaml:"prometheus"`
}

// Configures HTTP transport settings.
type HTTPConfig struct {
	Timeout         string `yaml:"timeout"`
	MaxIdleConns    int    `yaml:"max_idle_conns"`
	MaxConnsPerHost int    `yaml:"max_conns_per_host"`
}

type EngineConfig struct {
	VUs             int               `yaml:"vus"`
	Duration        string            `yaml:"duration"`
	MaxDuration     string            `yaml:"max_duration"`
	Iterations      *int              `yaml:"iterations"`
	StartAt         string            `yaml:"start_at"`
	Order           int               `yaml:"order"`
	Stages          []Stage           `yaml:"stages"`
	Thresholds      map[string]string `yaml:"thresholds"`
	Tags            map[string]string `yaml:"tags"`
	InsecureSkipTLS bool              `yaml:"insecure_skip_tls_verify"`
	Exporters       ExportersConfig   `yaml:"exporters"`
	HTTP            HTTPConfig        `yaml:"http"`
}

type Stage struct {
	Duration string `yaml:"duration"`
	Target   int    `yaml:"target"`
}

type ConfigVariable struct {
	VUs             *int
	Duration        *string
	MaxDuration     *string
	Iterations      *int
	StartAt         *string
	Order           *int
	InsecureSkipTLS *bool
}

type ASTConfig struct {
	ConfigVariable
	Stages     []Stage
	Thresholds map[string]string
	Tags       map[string]string
}

type EnvConfig struct {
	ConfigVariable
	HTTPTimeout   *string
	ExportJSON    *string
	ExportHTML    *string
	PrometheusOut *string
}

type CLIConfig struct {
	ConfigVariable
}

func DefaultEngineConfig() EngineConfig {
	return EngineConfig{
		VUs:        1,
		Iterations: nil,
		Order:      0,
		HTTP: HTTPConfig{
			Timeout:         "30s",
			MaxIdleConns:    10000,
			MaxConnsPerHost: 10000,
		},
	}
}
