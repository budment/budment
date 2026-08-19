package config

type EngineConfig struct {
	VUs             int               `yaml:"vus"`
	Duration        string            `yaml:"duration"`
	MaxDuration     string            `yaml:"max_duration"`
	Iterations      int               `yaml:"iterations"`
	StartAt         string            `yaml:"start_at"`
	Order           int               `yaml:"order"`
	Stages          []Stage           `yaml:"stages"`
	Thresholds      map[string]string `yaml:"thresholds"`
	Tags            map[string]string `yaml:"tags"`
	InsecureSkipTLS bool              `yaml:"insecure_skip_tls_verify"`
	AutoPlumb       bool              `yaml:"auto_plumb"`
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
	AutoPlumb       *bool
}

type ASTConfig struct {
	ConfigVariable
	Stages     []Stage
	Thresholds map[string]string
	Tags       map[string]string
}

type EnvConfig struct {
	ConfigVariable
}

type CLIConfig struct {
	ConfigVariable
}

type ParseConfig struct {
	SpecFile       string
	EndpointDir    string
	AIConfig       AIConfig
	EndpointConfig EndpointConfig
}

type EndpointConfig struct {
	IdentifierTokens   []string `yaml:"identifier_tokens"`
	WrapperNames       []string `yaml:"wrapper_names"`
	MinimumScore       float64  `yaml:"minimum_score"`
	SafetyMargin       float64  `yaml:"safety_margin"`
	JaroBoostThreshold float64  `yaml:"jaro_boost_threshold"`
	JaroPrefixSize     int      `yaml:"jaro_prefix_size"`
}

// DefaultEndpointConfig returns deterministic defaults for automatic Endpoint discovery.
func DefaultEndpointConfig() EndpointConfig {
	return EndpointConfig{
		IdentifierTokens:   []string{"id", "uuid", "code"},
		WrapperNames:       []string{"body", "data", "payload"},
		MinimumScore:       80.0,
		SafetyMargin:       15.0,
		JaroBoostThreshold: 0.7,
		JaroPrefixSize:     4,
	}
}

type AIConfig struct {
	BaseURL  string   `yaml:"base_url"`
	APIKey   string   `yaml:"api_key"`
	Model    string   `yaml:"model"`
	Features []string `yaml:"features"`
}

func (c *AIConfig) IsActive() bool {
	return c.BaseURL != "" && c.APIKey != "" && c.Model != ""
}

func DefaultEngineConfig() EngineConfig {
	return EngineConfig{
		VUs:        1,
		Iterations: 1,
		Order:      0,
		AutoPlumb:  false,
	}
}
