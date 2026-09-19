package engine

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/budment/budment/internal/config"
	"github.com/budment/budment/internal/metrics"
	"github.com/budment/budment/internal/planner"
	"github.com/budment/budment/internal/runner"
	"github.com/budment/budment/internal/runtime"
)

type Scenario struct {
	Name           string
	Config         config.EngineConfig
	Graph          *planner.Graph
	Pool           ExecutorPool
	Aggregator     *metrics.Aggregator
	BarrierManager *BarrierManager
	RunnerFactory  func() *runner.Manager
	LocalState     *LocalState
	GlobalState    *GlobalState
	Sink           runtime.MetricsSink
}

func NewScenario(name string, cfg config.EngineConfig, graph *planner.Graph, pool ExecutorPool, agg *metrics.Aggregator, sm *BarrierManager, runnerFactory func() *runner.Manager, ls *LocalState, gs *GlobalState, sink runtime.MetricsSink) *Scenario {
	return &Scenario{
		Name:           name,
		Config:         cfg,
		Graph:          graph,
		Pool:           pool,
		Aggregator:     agg,
		BarrierManager: sm,
		RunnerFactory:  runnerFactory,
		LocalState:     ls,
		GlobalState:    gs,
		Sink:           sink,
	}
}

func (s *Scenario) Run(parentCtx context.Context) {
	// Initialize Dual-Context for Graceful Shutdown
	hardCtx, hardCancel := context.WithCancel(parentCtx)
	defer hardCancel()

	softCtx, softCancel := context.WithCancel(parentCtx)
	defer softCancel()

	isUsingStages := len(s.Config.Stages) > 0
	hasDuration := s.Config.Duration != "" && s.Config.Duration != "0s"

	if hasDuration {
		dur, err := time.ParseDuration(s.Config.Duration)
		if err != nil {
			if s.Sink != nil {
				s.Sink.Log(0, "SCENARIO", "FATAL", fmt.Sprintf("Invalid duration config '%s': %v", s.Config.Duration, err))
			}
			return
		}

		go func() {
			select {
			case <-time.After(dur):
				softCancel() // Trigger graceful stop automatically
			case <-hardCtx.Done():
			}
		}()
	}

	var actualIters int
	if s.Config.Iterations != nil {
		actualIters = *s.Config.Iterations
	} else {
		if !isUsingStages && !hasDuration {
			actualIters = s.Config.VUs
			if actualIters <= 0 {
				actualIters = 1
			}
		} else {
			actualIters = 0
		}
	}

	var sharedIters int64

	// Shared flag signaling VUs to ramp down / terminate
	var activeTarget int64

	// PHASE 1: SETUP
	if len(s.Graph.Setup) > 0 {
		// Setup worker runs only once, so pass nil for activeTarget
		setupWorker := NewWorker(0, 0, s.Graph, s.Pool, s.Aggregator, s.BarrierManager, s.RunnerFactory, 1, s.Name, s.LocalState, s.GlobalState, s.Sink, &sharedIters, nil)
		setupWorker.Scope.Set("__VU_ID__", 0)
		setupWorker.Scope.Set("__ITER__", 0)
		setupWorker.Scope.Set("__SCENARIO__", s.Name)

		aborted := setupWorker.executeNodes(hardCtx, s.Graph.Setup)
		if aborted || setupWorker.Scope.IsAbortFlag() {
			if s.Sink != nil {
				s.Sink.Log(0, "SETUP", "FATAL", fmt.Sprintf("Setup failed for scenario '%s'. Aborting!", s.Name))
			}
			return
		}
		atomic.StoreInt64(&sharedIters, 0)
	}

	var wg sync.WaitGroup
	spawnFunc := func(slot int, id int) {
		wg.Add(1)
		go func(workerSlot int, workerID int) {
			defer wg.Done()
			w := NewWorker(workerID, workerSlot, s.Graph, s.Pool, s.Aggregator, s.BarrierManager, s.RunnerFactory, actualIters, s.Name, s.LocalState, s.GlobalState, s.Sink, &sharedIters, &activeTarget)
			w.Run(hardCtx, softCtx, func() {})
		}(slot, id)
	}

	// Monitor iteration limits to trigger soft cancellation
	if actualIters > 0 {
		go func() {
			ticker := time.NewTicker(50 * time.Millisecond)
			defer ticker.Stop()

			for {
				select {
				case <-hardCtx.Done():
					return
				case <-ticker.C:
					claimed := atomic.LoadInt64(&sharedIters)
					active := atomic.LoadInt64(&s.Aggregator.Metrics.ActiveVUs)
					if claimed >= int64(actualIters) && active == 0 {
						softCancel()
						return
					}
				}
			}
		}()
	}

	// PHASE 2: EXECUTION
	if !isUsingStages {
		scheduler := &ConstantVUScheduler{VUs: s.Config.VUs}
		scheduler.Run(softCtx, spawnFunc, &activeTarget)
	} else {
		scheduler := NewRampingScheduler(s.Config.Stages)
		scheduler.Run(softCtx, spawnFunc, &activeTarget)
	}

	// Ensure softCtx is cancelled when the scheduler completes its lifecycle
	softCancel()

	// Wait for Graceful Shutdown (Cooldown Phase)
	gracefulDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(gracefulDone)
	}()

	gracefulTimeout := 30 * time.Second
	select {
	case <-gracefulDone:
		// All VUs finished gracefully
	case <-time.After(gracefulTimeout):
		if s.Sink != nil {
			s.Sink.Log(0, "SCENARIO", "WARN", "Graceful timeout exceeded! Forcefully terminating active VUs.")
		}
		hardCancel()
		<-gracefulDone
	}

	// PHASE 3: TEARDOWN
	if len(s.Graph.Teardown) > 0 {
		teardownCtx, teardownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer teardownCancel()

		teardownWorker := NewWorker(0, 0, s.Graph, s.Pool, s.Aggregator, s.BarrierManager, s.RunnerFactory, 1, s.Name, s.LocalState, s.GlobalState, s.Sink, &sharedIters, nil)
		teardownWorker.Scope.Set("__VU_ID__", 0)
		teardownWorker.Scope.Set("__ITER__", 0)
		teardownWorker.Scope.Set("__SCENARIO__", s.Name)

		if s.Sink != nil {
			s.Sink.Log(0, "TEARDOWN", "INFO", fmt.Sprintf("Starting teardown for scenario '%s'", s.Name))
		}

		_ = teardownWorker.executeNodes(teardownCtx, s.Graph.Teardown)
	}
}
