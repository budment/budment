package engine

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vunas/blaster/internal/config"
	"github.com/vunas/blaster/internal/metrics"
	"github.com/vunas/blaster/internal/planner"
	"github.com/vunas/blaster/internal/runner"
	"github.com/vunas/blaster/internal/runtime"
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
	var ctx context.Context
	var cancel context.CancelFunc

	isUsingStages := len(s.Config.Stages) > 0
	hasDuration := s.Config.Duration != "" && s.Config.Duration != "0s"

	if hasDuration {
		dur, err := time.ParseDuration(s.Config.Duration)
		if err != nil {
			if s.Sink != nil {
				s.Sink.Log(0, "SCENARIO", "FATAL", fmt.Sprintf("Cấu hình duration sai '%s': %v", s.Config.Duration, err))
			}
			return
		}
		ctx, cancel = context.WithTimeout(parentCtx, dur)
	} else {
		ctx, cancel = context.WithCancel(parentCtx)
	}
	defer cancel()

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
	var activeTarget int32

	if len(s.Graph.Setup) > 0 {
		// Setup worker runs only once, so pass nil for activeTarget
		setupWorker := NewWorker(0, 0, s.Graph, s.Pool, s.Aggregator, s.BarrierManager, s.RunnerFactory, 1, s.Name, s.LocalState, s.GlobalState, s.Sink, &sharedIters, nil)
		setupWorker.Scope.Set("__VU_ID__", 0)
		setupWorker.Scope.Set("__ITER__", 0)
		setupWorker.Scope.Set("__SCENARIO__", s.Name)

		aborted := setupWorker.executeNodes(ctx, s.Graph.Setup)
		if aborted || setupWorker.Scope.IsAbortFlag() {
			if s.Sink != nil {
				s.Sink.Log(0, "SETUP", "FATAL", fmt.Sprintf("Setup thất bại trong kịch bản '%s'. Hủy bỏ scenario!", s.Name))
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
			w.Run(ctx, func() {})
		}(slot, id)
	}

	if !isUsingStages {
		scheduler := &ConstantVUScheduler{VUs: s.Config.VUs}
		scheduler.Start(ctx, spawnFunc, &activeTarget)
		wg.Wait()
		return
	}

	scheduler := NewRampingScheduler(s.Config.Stages)

	if actualIters > 0 {
		go func() {
			ticker := time.NewTicker(50 * time.Millisecond)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					claimed := atomic.LoadInt64(&sharedIters)
					active := atomic.LoadInt64(&s.Aggregator.Metrics.ActiveVUs)
					if claimed >= int64(actualIters) && active == 0 {
						cancel()
						return
					}
				}
			}
		}()
	}

	scheduler.Start(ctx, spawnFunc, &activeTarget)
	cancel()
	wg.Wait()
}
