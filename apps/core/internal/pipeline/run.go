package pipeline

import (
	"log/slog"

	"github.com/vunas/blaster/internal/config"
	"github.com/vunas/blaster/internal/engine"
	"github.com/vunas/blaster/internal/exporters"
	"github.com/vunas/blaster/internal/metrics"
	"github.com/vunas/blaster/internal/runner/http"
	"github.com/vunas/blaster/internal/runtime"
	"github.com/vunas/blaster/internal/runtime/goja"
)

type gojaPoolAdapter struct {
	pool *goja.Pool
}

func (a *gojaPoolAdapter) GetExecutor(scope runtime.Scope, local runtime.SharedState, workerID int, iteration int, scenario string) engine.HookExecutor {
	return a.pool.GetVM(scope, local, workerID, iteration, scenario)
}

func (a *gojaPoolAdapter) PutExecutor(executor engine.HookExecutor) {
	if inst, ok := executor.(*goja.VMInstance); ok {
		a.pool.PutVM(inst)
	}
}

type ExecutionApp struct {
	Director   *engine.Director
	Aggregator *metrics.Aggregator
}

func PrepareExecution(plan *PlanResult, cfg config.EngineConfig, log *slog.Logger, agg *metrics.Aggregator, sink runtime.MetricsSink, reporters []exporters.Reporter) (*ExecutionApp, error) {

	globalState := engine.NewGlobalState()
	localState := engine.NewLocalState()

	registry := goja.NewHookRegistry(string(plan.JSBundle))
	pool := goja.NewPool(registry, globalState, localState, sink)
	executorPool := &gojaPoolAdapter{pool: pool}

	barrierManager := engine.NewBarrierManager()
	httpClient := http.NewClient(cfg.InsecureSkipTLS, cfg.HTTP)

	var engineScenarios []*engine.Scenario

	for _, compScn := range plan.Scenarios {
		scenarioName := compScn.Name
		if scenarioName == "" {
			scenarioName = "default"
		}

		scnCfg := cfg

		if compScn.ASTConfig.VUs != nil {
			scnCfg.VUs = *compScn.ASTConfig.VUs
		}
		if compScn.ASTConfig.Duration != nil {
			scnCfg.Duration = *compScn.ASTConfig.Duration
		}
		if compScn.ASTConfig.Iterations != nil {
			scnCfg.Iterations = *compScn.ASTConfig.Iterations
		}
		if len(compScn.ASTConfig.Stages) > 0 {
			scnCfg.Stages = compScn.ASTConfig.Stages
		}
		if compScn.ASTConfig.StartAt != nil {
			scnCfg.StartAt = *compScn.ASTConfig.StartAt
		}
		if compScn.ASTConfig.Order != nil {
			scnCfg.Order = *compScn.ASTConfig.Order
		}
		scenario := engine.NewScenario(
			scenarioName,
			scnCfg,
			compScn.Graph,
			executorPool,
			agg,
			barrierManager,
			httpClient,
			localState,
			sink,
		)
		engineScenarios = append(engineScenarios, scenario)
	}

	director := engine.NewDirector(engineScenarios, cfg, agg, barrierManager, reporters)

	return &ExecutionApp{Director: director, Aggregator: agg}, nil
}
