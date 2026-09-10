package pipeline

import (
	"github.com/budment/budment/internal/config"
	"github.com/budment/budment/internal/engine"
	"github.com/budment/budment/internal/exporters"
	"github.com/budment/budment/internal/metrics"
	"github.com/budment/budment/internal/runner"
	"github.com/budment/budment/internal/runner/http"
	"github.com/budment/budment/internal/runtime"
	"github.com/budment/budment/internal/runtime/goja"
)

type gojaPoolAdapter struct {
	pool *goja.Pool
}

func (a *gojaPoolAdapter) GetExecutor(scope runtime.VUContext, local runtime.SharedState, workerID int, iteration int, scenario string) engine.HookExecutor {
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

func PrepareExecution(plan *PlanResult, cfg config.EngineConfig, agg *metrics.Aggregator, sink runtime.MetricsSink, reporters []exporters.Reporter) (*ExecutionApp, error) {
	globalState := engine.NewGlobalState()

	registry := goja.NewHookRegistry(string(plan.JSBundle))
	pool := goja.NewPool(registry, globalState, sink)
	executorPool := &gojaPoolAdapter{pool: pool}

	barrierManager := engine.NewBarrierManager()
	sharedTransport := http.NewSharedTransport(cfg.InsecureSkipTLS)

	managerFactory := func() *runner.Manager {
		m := runner.NewManager()
		m.Register("http", http.NewRunner(sharedTransport))
		return m
	}

	engineScenarios := make([]*engine.Scenario, 0, len(plan.Scenarios))

	for i := range plan.Scenarios {
		compScn := plan.Scenarios[i]
		localState := engine.NewLocalState()
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
			scnCfg.Iterations = compScn.ASTConfig.Iterations
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
			managerFactory,
			localState,
			globalState,
			sink,
		)
		engineScenarios = append(engineScenarios, scenario)
	}

	director := engine.NewDirector(engineScenarios, cfg, agg, barrierManager)

	return &ExecutionApp{Director: director, Aggregator: agg}, nil
}
