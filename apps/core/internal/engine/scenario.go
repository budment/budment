package engine

import (
	"context"
	"sync"
	"time"

	"github.com/vunas/blaster/internal/config"
	"github.com/vunas/blaster/internal/metrics"
	"github.com/vunas/blaster/internal/planner"
	"github.com/vunas/blaster/internal/runner/http"
	"github.com/vunas/blaster/internal/runtime"
)

type Scenario struct {
	Name           string
	Config         config.EngineConfig
	Graph          *planner.Graph
	Pool           ExecutorPool
	Aggregator     *metrics.Aggregator
	BarrierManager *BarrierManager
	HTTPClient     *http.Client
	LocalState     *LocalState
	Sink           runtime.MetricsSink
}

func NewScenario(name string, cfg config.EngineConfig, graph *planner.Graph, pool ExecutorPool, agg *metrics.Aggregator, sm *BarrierManager, client *http.Client, ls *LocalState, sink runtime.MetricsSink) *Scenario {
	return &Scenario{
		Name:           name,
		Config:         cfg,
		Graph:          graph,
		Pool:           pool,
		Aggregator:     agg,
		BarrierManager: sm,
		HTTPClient:     client,
		LocalState:     ls,
		Sink:           sink,
	}
}

func (s *Scenario) Run(parentCtx context.Context) {
	var ctx context.Context
	var cancel context.CancelFunc

	actualIters := s.Config.Iterations

	if s.Config.Duration != "" && s.Config.Duration != "0s" {
		dur, _ := time.ParseDuration(s.Config.Duration)
		ctx, cancel = context.WithTimeout(parentCtx, dur)
	} else {
		ctx, cancel = context.WithCancel(parentCtx)
	}
	defer cancel()

	var sharedIters int64

	// Shared flag signaling VUs to ramp down / terminate
	var activeTarget int32

	if len(s.Graph.Setup) > 0 {
		// Setup worker runs only once, so pass nil for activeTarget
		setupWorker := NewWorker(0, s.Graph, s.Pool, s.Aggregator, s.BarrierManager, s.HTTPClient, 1, s.Name, s.LocalState, s.Sink, &sharedIters, nil)
		setupWorker.executeNodes(ctx, s.Graph.Setup)
	}

	var scheduler Scheduler
	isUsingStages := len(s.Config.Stages) > 0

	if isUsingStages {
		scheduler = NewRampingScheduler(s.Config.Stages)
	} else {
		scheduler = &ConstantVUScheduler{VUs: s.Config.VUs}
	}

	var wg sync.WaitGroup
	spawnFunc := func(id int) {
		wg.Add(1)
		go func(workerID int) {
			w := NewWorker(workerID, s.Graph, s.Pool, s.Aggregator, s.BarrierManager, s.HTTPClient, actualIters, s.Name, s.LocalState, s.Sink, &sharedIters, &activeTarget)
			w.Run(ctx, wg.Done)
		}(id)
	}
	scheduler.Start(ctx, spawnFunc, &activeTarget)

	if isUsingStages {
		cancel()
	}

	wg.Wait()
}
