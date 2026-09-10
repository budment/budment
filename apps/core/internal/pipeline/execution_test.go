package pipeline

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/budment/budment/internal/config"
	"github.com/budment/budment/internal/engine"
	"github.com/budment/budment/internal/metrics"
	"github.com/budment/budment/internal/planner"
	"github.com/budment/budment/internal/runtime/goja"
)

type stubExecutionSink struct{}

func (s *stubExecutionSink) Log(workerID int, nodeID string, level string, msg string) {}
func (s *stubExecutionSink) RecordEvent(workerID int, nodeID string, eventType string, reason string) {
}
func (s *stubExecutionSink) Tag(workerID int, key string, value string)                        {}
func (s *stubExecutionSink) RecordCustom(workerID int, mType string, name string, val float64) {}

func TestPrepareExecution_LifecycleAndDirectorRun(t *testing.T) {
	iters := 4
	astCfg := config.ASTConfig{
		ConfigVariable: config.ConfigVariable{
			Iterations: &iters,
		},
	}

	mockGraph := &planner.Graph{
		Execution: []planner.ExecutableNode{
			&planner.SleepNode{ID: "sleep_node", Duration: 2 * time.Millisecond},
		},
	}

	plan := &PlanResult{
		Scenarios: []CompiledScenario{
			{
				Name:      "SmokeExecutionScenario",
				Graph:     mockGraph,
				ASTConfig: astCfg,
			},
		},
		ASTConfig: astCfg,
		JSBundle:  []byte(`// empty mock bundle`),
	}

	baseCfg := config.EngineConfig{
		VUs:         2,
		MaxDuration: "1s",
	}

	agg := metrics.NewAggregator(500, nil)
	sink := &stubExecutionSink{}

	app, err := PrepareExecution(plan, baseCfg, agg, sink, nil)
	if err != nil {
		t.Fatalf("PrepareExecution failed: %v", err)
	}

	if app == nil || app.Director == nil {
		t.Fatal("PrepareExecution returned nil ExecutionApp or Director")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 800*time.Millisecond)
	defer cancel()

	runDone := make(chan error, 1)
	go func() {
		runDone <- app.Director.Run()
	}()

	select {
	case err := <-runDone:
		if err != nil {
			t.Fatalf("director execution returned unexpected error: %v", err)
		}
	case <-ctx.Done():
		t.Fatal("director execution deadlocked or exceeded context deadline")
	}

	activeVUs := atomic.LoadInt64(&agg.Metrics.ActiveVUs)
	if activeVUs != 0 {
		t.Errorf("active VUs leaked after run: expected 0, got %d", activeVUs)
	}

	_, completedIters, _, _, _, _, _, _ := agg.Metrics.Snapshot()
	if completedIters != int64(iters) {
		t.Errorf("iteration count mismatch: expected %d, got %d", iters, completedIters)
	}
}

func TestPrepareExecution_ASTConfigOverrides(t *testing.T) {
	overrideVUs := 8
	overrideDuration := "300ms"
	overrideOrder := 3
	overrideStartAt := "50ms"
	stages := []config.Stage{{Duration: "100ms", Target: 8}}

	astCfg := config.ASTConfig{
		ConfigVariable: config.ConfigVariable{
			VUs:      &overrideVUs,
			Duration: &overrideDuration,
			Order:    &overrideOrder,
			StartAt:  &overrideStartAt,
		},
		Stages: stages,
	}

	mockPlan := &PlanResult{
		Scenarios: []CompiledScenario{
			{
				Name:      "OverriddenScenario",
				Graph:     &planner.Graph{},
				ASTConfig: astCfg,
			},
		},
		ASTConfig: astCfg,
		JSBundle:  []byte(`// mock bundle`),
	}

	baseEngineCfg := config.EngineConfig{
		VUs:      1,
		Duration: "10s",
		Order:    1,
	}

	agg := metrics.NewAggregator(100, nil)
	sink := &stubExecutionSink{}

	app, err := PrepareExecution(mockPlan, baseEngineCfg, agg, sink, nil)
	if err != nil {
		t.Fatalf("PrepareExecution failed: %v", err)
	}

	if len(app.Director.Scenarios) != 1 {
		t.Fatalf("registered scenarios count mismatch: expected 1, got %d", len(app.Director.Scenarios))
	}

	targetScn := app.Director.Scenarios[0]

	if targetScn.Config.VUs != overrideVUs {
		t.Errorf("VUs override failed: expected %d, got %d", overrideVUs, targetScn.Config.VUs)
	}
	if targetScn.Config.Duration != overrideDuration {
		t.Errorf("Duration override failed: expected %s, got %s", overrideDuration, targetScn.Config.Duration)
	}
	if targetScn.Config.Order != overrideOrder {
		t.Errorf("Order override failed: expected %d, got %d", overrideOrder, targetScn.Config.Order)
	}
	if targetScn.Config.StartAt != overrideStartAt {
		t.Errorf("StartAt override failed: expected %s, got %s", overrideStartAt, targetScn.Config.StartAt)
	}
	if len(targetScn.Config.Stages) != len(stages) {
		t.Errorf("Stages override count mismatch: expected %d, got %d", len(stages), len(targetScn.Config.Stages))
	}
}

func TestGojaPoolAdapter_ExecutorAcquisitionAndRelease(t *testing.T) {
	jsCode := `
		globalThis.HookRegistry = {
			hooks: new Map(),
			register: function(id, fn) { this.hooks.set(id, fn); }
		};
		globalThis.HookRegistry.register("hook_ping", function() {
			return "pong";
		});
	`
	registry := goja.NewHookRegistry(jsCode)
	globalState := engine.NewGlobalState()
	sink := &stubExecutionSink{}

	pool := goja.NewPool(registry, globalState, sink)
	adapter := &gojaPoolAdapter{pool: pool}

	scope := engine.NewWorkerScope(nil, globalState)
	localState := engine.NewLocalState()

	executor := adapter.GetExecutor(scope, localState, 1, 0, "test_scenario")
	if executor == nil {
		t.Fatal("expected non-nil HookExecutor from gojaPoolAdapter")
	}

	resStr, err := executor.EvaluateString("hook_ping")
	if err != nil {
		t.Fatalf("failed to evaluate hook via executor: %v", err)
	}
	if resStr != "pong" {
		t.Errorf("executor evaluation mismatch: expected 'pong', got '%s'", resStr)
	}

	adapter.PutExecutor(executor)
}
