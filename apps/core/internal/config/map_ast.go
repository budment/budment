package config

import (
	"maps"

	ast "github.com/budment/budment/internal/planner/pb"
)

func Ptr[T any](v T) *T {
	return &v
}

func MapScenarioConfigToAST(astGraph *ast.Scenario, astCfg *ASTConfig) {
	if astGraph == nil || astGraph.Config == nil {
		return
	}
	cfg := astGraph.Config
	if cfg.GetVus() > 0 {
		astCfg.VUs = Ptr(int(cfg.GetVus()))
	}
	if cfg.GetDuration() != "" {
		astCfg.Duration = Ptr(cfg.GetDuration())
	}
	if cfg.GetMaxDuration() != "" {
		astCfg.MaxDuration = Ptr(cfg.GetMaxDuration())
	}
	if cfg.GetIterations() > 0 {
		astCfg.Iterations = Ptr(int(cfg.GetIterations()))
	}
	if cfg.GetStartAt() != "" {
		astCfg.StartAt = Ptr(cfg.GetStartAt())
	}
	if cfg.Order != nil {
		astCfg.Order = Ptr(int(cfg.GetOrder()))
	}
	if cfg.InsecureSkipTlsVerify != nil {
		astCfg.InsecureSkipTLS = Ptr(cfg.GetInsecureSkipTlsVerify())
	}
	if len(cfg.Stages) > 0 {
		astCfg.Stages = make([]Stage, len(cfg.Stages))
		for i, s := range cfg.Stages {
			if s != nil {
				astCfg.Stages[i] = Stage{
					Duration: s.GetDuration(),
					Target:   int(s.GetTarget()),
				}
			}
		}
	}
	if len(cfg.Thresholds) > 0 {
		astCfg.Thresholds = make(map[string]string, len(cfg.Thresholds))
		maps.Copy(astCfg.Thresholds, cfg.Thresholds)
	}
	if len(cfg.Tags) > 0 {
		astCfg.Tags = make(map[string]string, len(cfg.Tags))
		maps.Copy(astCfg.Tags, cfg.Tags)
	}
}
