package config

// Default -> YAML -> AST (Script) -> ENV -> CLI
func MergeEngineConfig(yamlConfig EngineConfig, astConfig ASTOverrides, env OverrideConfig, cli OverrideConfig) EngineConfig {
	// default config
	final := yamlConfig
	mergeASTOverrides(&final, astConfig)
	applyEnvOverrides(&final, env)
	applyCLIOverrides(&final, cli)

	return final
}

func mergeASTOverrides(dst *EngineConfig, src ASTOverrides) {
	if src.VUs != nil {
		dst.VUs = *src.VUs
	}
	if src.Duration != nil {
		dst.Duration = *src.Duration
	}
	if src.MaxDuration != nil {
		dst.MaxDuration = *src.MaxDuration
	}
	if src.Iterations != nil {
		dst.Iterations = *src.Iterations
	}
	if src.StartAt != nil {
		dst.StartAt = *src.StartAt
	}

	if src.Order != nil {
		dst.Order = *src.Order
	}

	if len(src.Stages) > 0 {
		dst.Stages = src.Stages
	}

	dst.Thresholds = mergeMaps(dst.Thresholds, src.Thresholds)
	dst.Tags = mergeMaps(dst.Tags, src.Tags)

	if src.InsecureSkipTLS != nil {
		dst.InsecureSkipTLS = *src.InsecureSkipTLS
	}
	if src.AutoPlumb != nil {
		dst.AutoPlumb = *src.AutoPlumb
	}
}

func applyEnvOverrides(dst *EngineConfig, env OverrideConfig) {
	if env.VUs != nil {
		dst.VUs = *env.VUs
	}
	if env.Duration != nil {
		dst.Duration = *env.Duration
	}
	if env.MaxDuration != nil {
		dst.MaxDuration = *env.MaxDuration
	}
	if env.Iterations != nil {
		dst.Iterations = *env.Iterations
	}
	if env.StartAt != nil {
		dst.StartAt = *env.StartAt
	}
	if env.InsecureSkipTLS != nil {
		dst.InsecureSkipTLS = *env.InsecureSkipTLS
	}
	if env.AutoPlumb != nil {
		dst.AutoPlumb = *env.AutoPlumb
	}
}

func applyCLIOverrides(dst *EngineConfig, cli OverrideConfig) {
	if cli.VUs != nil {
		dst.VUs = *cli.VUs
	}
	if cli.Duration != nil {
		dst.Duration = *cli.Duration
	}
	if cli.MaxDuration != nil {
		dst.MaxDuration = *cli.MaxDuration
	}
	if cli.Iterations != nil {
		dst.Iterations = *cli.Iterations
	}
	if cli.StartAt != nil {
		dst.StartAt = *cli.StartAt
	}
	if cli.Order != nil {
		dst.Order = *cli.Order
	}
	if cli.InsecureSkipTLS != nil {
		dst.InsecureSkipTLS = *cli.InsecureSkipTLS
	}
	if cli.AutoPlumb != nil {
		dst.AutoPlumb = *cli.AutoPlumb
	}
}

func mergeMaps(base, src map[string]string) map[string]string {
	if base == nil && src == nil {
		return nil
	}
	result := make(map[string]string)
	for k, v := range base {
		result[k] = v
	}
	for k, v := range src {
		result[k] = v
	}
	return result
}
