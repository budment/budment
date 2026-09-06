package config

import "maps"

func MergeEngineConfig(yamlConfig EngineConfig, astConfig ASTConfig, env EnvConfig, cli CLIConfig) EngineConfig {
	final := yamlConfig

	applyConfigVariable(&final, astConfig.ConfigVariable)
	if len(astConfig.Stages) > 0 {
		final.Stages = astConfig.Stages
	}
	final.Thresholds = mergeMaps(final.Thresholds, astConfig.Thresholds)
	final.Tags = mergeMaps(final.Tags, astConfig.Tags)

	applyConfigVariable(&final, env.ConfigVariable)
	applyEnvExtraOverrides(&final, env)

	applyConfigVariable(&final, cli.ConfigVariable)

	return final
}

func applyConfigVariable(dst *EngineConfig, src ConfigVariable) {
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
		dst.Iterations = src.Iterations
	}
	if src.StartAt != nil {
		dst.StartAt = *src.StartAt
	}
	if src.Order != nil {
		dst.Order = *src.Order
	}
	if src.InsecureSkipTLS != nil {
		dst.InsecureSkipTLS = *src.InsecureSkipTLS
	}
}

func applyEnvExtraOverrides(dst *EngineConfig, env EnvConfig) {
	if env.HTTPTimeout != nil {
		dst.HTTP.Timeout = *env.HTTPTimeout
	}
	if env.ExportJSON != nil {
		dst.Exporters.JSON = *env.ExportJSON
	}
	if env.ExportHTML != nil {
		dst.Exporters.HTML = *env.ExportHTML
	}
	if env.PrometheusOut != nil {
		dst.Exporters.Prometheus = *env.PrometheusOut
	}
}

func mergeMaps(base, src map[string]string) map[string]string {
	if len(base) == 0 && len(src) == 0 {
		return nil
	}
	res := make(map[string]string, len(base)+len(src))
	maps.Copy(res, base)
	maps.Copy(res, src)
	return res
}
