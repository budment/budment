package engine

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/vunas/blaster/internal/config"
	"github.com/vunas/blaster/internal/metrics"
	"github.com/vunas/blaster/internal/planner"
	"github.com/vunas/blaster/internal/runner"
)

func TestDirector_ScenarioOrderAndGracefulShutdown(t *testing.T) {
	agg := metrics.NewAggregator(100)
	pool := &dummyExecutorPool{}
	runnerFac := func() *runner.Manager { return nil }

	graph := &planner.Graph{
		Execution: []planner.ExecutableNode{
			&planner.SleepNode{ID: "sleep", Duration: 5 * time.Millisecond},
		},
	}

	// Sequential execution order: Scn1 (Order 1) -> Scn2 (Order 2)
	scn1 := NewScenario("Scn1", config.EngineConfig{Order: 1, VUs: 1, Iterations: intPtr(2)}, graph, pool, agg, nil, runnerFac, nil, nil, nil)
	scn2 := NewScenario("Scn2", config.EngineConfig{Order: 2, VUs: 1, Iterations: intPtr(3)}, graph, pool, agg, nil, runnerFac, nil, nil, nil)

	director := NewDirector([]*Scenario{scn1, scn2}, config.EngineConfig{}, agg, nil, nil)

	err := director.Run()
	if err != nil {
		t.Fatalf("Director run failed: %v", err)
	}

	// Total iterations: 2 (from Scn1) + 3 (from Scn2) = 5
	_, iters, _, _, _, _, _, _ := agg.Metrics.Snapshot()
	if iters != 5 {
		t.Errorf("total iterations mismatch across sequential scenarios: expected 5, got %d", iters)
	}
}

func TestDirector_MaxDurationInterruption(t *testing.T) {
	agg := metrics.NewAggregator(100)
	pool := &dummyExecutorPool{}
	runnerFac := func() *runner.Manager { return nil }

	// Infinite duration scenario
	graph := &planner.Graph{
		Execution: []planner.ExecutableNode{
			&planner.SleepNode{ID: "sleep", Duration: 100 * time.Millisecond},
		},
	}

	scn1 := NewScenario("ScnInf", config.EngineConfig{Order: 1, VUs: 10, Duration: "0s"}, graph, pool, agg, nil, runnerFac, nil, nil, nil)

	// Enforce global 30ms hard cutoff
	baseCfg := config.EngineConfig{MaxDuration: "30ms"}
	director := NewDirector([]*Scenario{scn1}, baseCfg, agg, nil, nil)

	start := time.Now()
	err := director.Run()
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify hard cutoff aborted all active workers promptly
	if elapsed > 100*time.Millisecond {
		t.Errorf("director execution exceeded timeout threshold (%v): MaxDuration abort failed", elapsed)
	}

	activeVUs := atomic.LoadInt64(&agg.Metrics.ActiveVUs)
	if activeVUs != 0 {
		t.Errorf("active VUs leaked after abort: expected 0, got %d", activeVUs)
	}
}
