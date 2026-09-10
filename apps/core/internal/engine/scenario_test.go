package engine

import (
	"context"
	"testing"
	"time"

	"github.com/budment/budment/internal/config"
	"github.com/budment/budment/internal/metrics"
	"github.com/budment/budment/internal/planner"
	"github.com/budment/budment/internal/runner"
	"github.com/budment/budment/internal/runtime"
)

// dummyExecutorPool avoids nil-pointer panics in test graphs containing SleepNode.
type dummyExecutorPool struct{}

func (d *dummyExecutorPool) GetExecutor(scope runtime.VUContext, local runtime.SharedState, workerID int, iteration int, scenario string) HookExecutor {
	return nil
}
func (d *dummyExecutorPool) PutExecutor(executor HookExecutor) {}

func intPtr(v int) *int {
	return &v
}

func TestScenario_Run_ConstantVUs_And_Iterations(t *testing.T) {
	agg := metrics.NewAggregator(100, nil)
	ctxAgg, cancelAgg := context.WithCancel(context.Background())
	aggDone := make(chan struct{})
	go agg.Run(ctxAgg, aggDone)

	// Pipeline: 1 Setup node and 1 Execution node
	graph := &planner.Graph{
		Setup: []planner.ExecutableNode{
			&planner.SleepNode{ID: "setup", Duration: 1 * time.Millisecond},
		},
		Execution: []planner.ExecutableNode{
			&planner.SleepNode{ID: "exec", Duration: 2 * time.Millisecond},
		},
	}

	// 2 VUs sharing 4 total iterations
	cfg := config.EngineConfig{
		VUs:        2,
		Iterations: new(4),
	}

	scn := NewScenario("TestScn", cfg, graph, &dummyExecutorPool{}, agg, nil, func() *runner.Manager { return nil }, nil, nil, nil)
	scn.Run(context.Background())

	cancelAgg()
	<-aggDone // Wait for metrics flush

	_, iters, _, _, _, activeVUs, _, _ := agg.Metrics.Snapshot()
	if iters != 4 {
		t.Errorf("total iterations mismatch: expected 4, got %d", iters)
	}
	if activeVUs != 0 {
		t.Errorf("active VUs leaked after scenario completion: expected 0, got %d", activeVUs)
	}
}

func TestScenario_Run_DurationTimeout(t *testing.T) {
	agg := metrics.NewAggregator(100, nil)
	ctxAgg, cancelAgg := context.WithCancel(context.Background())
	aggDone := make(chan struct{})
	go agg.Run(ctxAgg, aggDone)

	graph := &planner.Graph{
		Execution: []planner.ExecutableNode{
			&planner.SleepNode{ID: "exec", Duration: 10 * time.Millisecond},
		},
	}

	// Infinite-duration scenario forced to stop after 30ms
	cfg := config.EngineConfig{
		VUs:      5,
		Duration: "30ms",
	}

	scn := NewScenario("TimeoutScn", cfg, graph, &dummyExecutorPool{}, agg, nil, func() *runner.Manager { return nil }, nil, nil, nil)

	start := time.Now()
	scn.Run(context.Background())
	elapsed := time.Since(start)

	cancelAgg()
	<-aggDone

	if elapsed > 100*time.Millisecond {
		t.Errorf("scenario exceeded timeout threshold (%v): duration cutoff failed", elapsed)
	}
	_, _, _, _, _, activeVUs, _, _ := agg.Metrics.Snapshot()
	if activeVUs != 0 {
		t.Errorf("active VUs leaked: expected 0, got %d", activeVUs)
	}
}
