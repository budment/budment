package pipeline

import (
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/vunas/blaster/internal/config"
	"github.com/vunas/blaster/internal/planner"
	"github.com/vunas/blaster/internal/planner/bundler"
)

type CompiledScenario struct {
	Name      string
	Graph     *planner.Graph
	ASTConfig config.ASTConfig
	RawJSON   json.RawMessage
}

type PlanResult struct {
	Scenarios  []CompiledScenario
	RawASTJSON []byte
	JSBundle   []byte
	ASTConfig  config.ASTConfig // Global config extracted from the first scenario
}

func BuildPlan(scriptPath string) (*PlanResult, error) {
	// Bundle TS/JS script into a single JS payload
	jsBundle, err := bundler.BundleInMemory(scriptPath)
	if err != nil {
		return nil, fmt.Errorf("esbuild failed: %w", err)
	}

	// Evaluate bundle to generate AST scenarios
	eval := planner.NewEvaluator()
	astScenarios, err := eval.Evaluate(string(jsBundle))
	if err != nil {
		return nil, fmt.Errorf("ast evaluation failed: %w", err)
	}

	compiler := planner.NewGraphCompiler()

	var compiledScenarios []CompiledScenario
	var allRaw []json.RawMessage

	marshaller := protojson.MarshalOptions{Multiline: true, Indent: "  "}

	for _, astGraph := range astScenarios {
		astCfg := config.ASTConfig{}
		if astGraph.Config != nil {
			config.MapScenarioConfigToAST(astGraph, &astCfg)
		}

		rawJSON, err := marshaller.Marshal(astGraph)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal AST graph to JSON: %w", err)
		}
		allRaw = append(allRaw, rawJSON)

		// Compile AST into an execution graph
		executionGraph, err := compiler.Compile(astGraph)
		if err != nil {
			return nil, fmt.Errorf("graph compilation failed for scenario %s: %w", astGraph.Name, err)
		}

		compiledScenarios = append(compiledScenarios, CompiledScenario{
			Name:      astGraph.Name,
			Graph:     executionGraph,
			ASTConfig: astCfg,
			RawJSON:   rawJSON,
		})
	}

	finalRawJSON, _ := json.MarshalIndent(allRaw, "", "  ")

	// Fallback to the first scenario's config as global config
	var mainASTConfig config.ASTConfig
	if len(compiledScenarios) > 0 {
		mainASTConfig = compiledScenarios[0].ASTConfig
	}

	return &PlanResult{
		Scenarios:  compiledScenarios,
		RawASTJSON: finalRawJSON,
		JSBundle:   jsBundle,
		ASTConfig:  mainASTConfig,
	}, nil
}
