package engine

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"sync"
	"syscall"
	"time"

	"github.com/vunas/blaster/internal/config"
	"github.com/vunas/blaster/internal/exporters"
	"github.com/vunas/blaster/internal/metrics"
)

type Director struct {
	Scenarios      []*Scenario
	BaseConfig     config.EngineConfig
	Aggregator     *metrics.Aggregator
	BarrierManager *BarrierManager
	Reporters      []exporters.Reporter
}

func NewDirector(scenarios []*Scenario, baseCfg config.EngineConfig, agg *metrics.Aggregator, sm *BarrierManager, reporters []exporters.Reporter) *Director {
	return &Director{Scenarios: scenarios, BaseConfig: baseCfg, Aggregator: agg, BarrierManager: sm, Reporters: reporters}
}

func (d *Director) Run() error {
	workerCtx, cancelWorkers := context.WithCancel(context.Background())
	defer func() { cancelWorkers() }()

	if d.BaseConfig.MaxDuration != "" {
		if dur, err := time.ParseDuration(d.BaseConfig.MaxDuration); err == nil {
			workerCtx, cancelWorkers = context.WithTimeout(workerCtx, dur)
		} else {
			fmt.Printf("[Director] Failed to parse MaxDuration: %v\n", err)
		}
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	go func() {
		select {
		case <-sigCh:
			cancelWorkers()
		case <-workerCtx.Done():
			return
		}
	}()

	aggCtx, cancelAgg := context.WithCancel(context.Background())
	defer cancelAgg()

	aggDone := make(chan struct{})
	go d.Aggregator.Run(aggCtx, aggDone)

	groups := make(map[int][]*Scenario)
	var orders []int

	for _, s := range d.Scenarios {
		order := s.Config.Order
		if _, exists := groups[order]; !exists {
			orders = append(orders, order)
		}
		groups[order] = append(groups[order], s)
	}

	sort.Ints(orders)
	assertMgr := metrics.NewAssertionManager(d.BaseConfig.Thresholds)

	for _, order := range orders {
		scenariosInGroup := groups[order]
		var groupWg sync.WaitGroup

		for _, s := range scenariosInGroup {
			groupWg.Add(1)
			go func(scenario *Scenario) {
				defer groupWg.Done()

				if scenario.Config.StartAt != "" {
					if delay, err := time.ParseDuration(scenario.Config.StartAt); err == nil {
						select {
						case <-time.After(delay):
						case <-workerCtx.Done():
							return
						}
					}
				}

				scenario.Run(workerCtx)
			}(s)
		}

		groupDone := make(chan struct{})
		go func() {
			groupWg.Wait()
			close(groupDone)
		}()

		select {
		case <-groupDone:
		case <-workerCtx.Done():
			<-groupDone
			goto FINISH
		}
	}

FINISH:
	cancelWorkers()
	cancelAgg()
	<-aggDone

	for _, rep := range d.Reporters {
		if err := rep.Export(d.Aggregator.Metrics, assertMgr); err != nil {
			fmt.Printf("Exporter Warning: %v\n", err)
		}
	}
	return nil
}
