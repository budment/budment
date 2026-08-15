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
	IdentifierTokens   []string `yaml:"identifier_tokens"`
	WrapperNames       []string `yaml:"wrapper_names"`
	MinimumScore       float64  `yaml:"minimum_score"`
	SafetyMargin       float64  `yaml:"safety_margin"`
	JaroBoostThreshold float64  `yaml:"jaro_boost_threshold"`
	JaroPrefixSize     int      `yaml:"jaro_prefix_size"`
}

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

// WorkflowConfig holds the paths and parameters required for the workflow generation pipeline.
type WorkflowConfig struct {
	SpecFile    string
	EndpointDir string
	WorkflowDir string

	// Generation dictates how payloads are expanded (happy path, boundary, invalid).
	// This will be heavily utilized by the Planner module.
	Generation GenerationProfile `yaml:"generation"`
}

// GenerationProfile defines the testing boundaries and payload expansion strategies.
type GenerationProfile struct {
	Profile string         `yaml:"profile"` // e.g., "normal", "aggressive", "smoke"
	Integer IntegerProfile `yaml:"integer"`
	String  StringProfile  `yaml:"string"`
}

// IntegerProfile defines the number of payload variations for integer fields.
type IntegerProfile struct {
	Happy    int `yaml:"happy"`
	Boundary int `yaml:"boundary"`
	Invalid  int `yaml:"invalid"`
}

// StringProfile defines the number of payload variations for string fields.
type StringProfile struct {
	Happy    int `yaml:"happy"`
	Security int `yaml:"security"`
	Invalid  int `yaml:"invalid"`
}

// DefaultWorkflowConfig provides sensible defaults for the workflow engine.
// It generates a conservative amount of test cases (Smoke test style) by default.
func DefaultWorkflowConfig() WorkflowConfig {
	return WorkflowConfig{
		SpecFile:    "openapi.yaml",
		EndpointDir: "endpoint",
		WorkflowDir: "workflow",
		Generation: GenerationProfile{
			Profile: "normal",
			Integer: IntegerProfile{
				Happy:    1,
				Boundary: 2,
				Invalid:  1,
			},
			String: StringProfile{
				Happy:    1,
				Security: 2,
				Invalid:  1,
			},
		},
	}
}
