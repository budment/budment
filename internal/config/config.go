package config

// ParseConfig holds the isolated configuration for the Parse pipeline.
type ParseConfig struct {
	SpecFile       string
	EndpointDir    string
	AIConfig       AIConfig
	EndpointConfig EndpointConfig
}

// EndpointConfig holds tuning parameters for endpoint generation.
type EndpointConfig struct {
	IdentifierTokens []string `yaml:"identifier_tokens"`
	WrapperNames     []string `yaml:"wrapper_names"`
	MinimumScore     float64  `yaml:"minimum_score"`
	SafetyMargin     float64  `yaml:"safety_margin"`
}

func DefaultEndpointConfig() EndpointConfig {
	return EndpointConfig{
		IdentifierTokens: []string{"id", "uuid", "code"},
		WrapperNames:     []string{"body", "data", "payload"},
		MinimumScore:     80.0,
		SafetyMargin:     15.0,
	}
}

type DiscoveryConfig struct {
	IdentifierTokens []string `yaml:"identifier_tokens"`
	WrapperNames     []string `yaml:"wrapper_names"`
}

type ResolutionConfig struct {
	MinimumScore float64 `yaml:"minimum_score"`
	SafetyMargin float64 `yaml:"safety_margin"`
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
