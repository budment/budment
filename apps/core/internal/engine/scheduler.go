package engine

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/vunas/blaster/internal/config"
)

type Scheduler interface {
	Start(ctx context.Context, workerFactory func(id int), activeTarget *int32)
}

type ConstantVUScheduler struct{ VUs int }

func (s *ConstantVUScheduler) Start(ctx context.Context, spawnWorker func(id int), activeTarget *int32) {
	atomic.StoreInt32(activeTarget, int32(s.VUs))
	for i := 1; i <= s.VUs; i++ {
		spawnWorker(i)
	}
}

type RampingScheduler struct{ Stages []config.Stage }

func NewRampingScheduler(stages []config.Stage) *RampingScheduler {
	return &RampingScheduler{Stages: stages}
}

func (s *RampingScheduler) Start(ctx context.Context, spawnWorker func(id int), activeTarget *int32) {
	currentVUs := 0
	for _, stage := range s.Stages {
		duration, err := time.ParseDuration(stage.Duration)
		if err != nil {
			continue
		}

		targetVUs := stage.Target

		// Update target to let current workers scale down if target < current
		atomic.StoreInt32(activeTarget, int32(targetVUs))

		if targetVUs > currentVUs {
			diff := targetVUs - currentVUs
			tickDuration := 10 * time.Millisecond
			totalTicks := int(duration / tickDuration)
			if totalTicks <= 0 {
				totalTicks = 1
			}

			vusPerTick := float64(diff) / float64(totalTicks)
			var spawned float64 = 0

			ticker := time.NewTicker(tickDuration)

			// Worker IDs continue sequentially from the current count (e.g., ramping 20 -> 50 starts IDs at 21)
			workerID := currentVUs

			for i := 0; i < totalTicks; i++ {
				select {
				case <-ctx.Done():
					ticker.Stop()
					return
				case <-ticker.C:
					spawned += vusPerTick
					toSpawnNow := int(spawned) - (workerID - currentVUs)

					for j := 0; j < toSpawnNow; j++ {
						workerID++
						spawnWorker(workerID)
					}
				}
			}
			ticker.Stop()

			for workerID < targetVUs {
				workerID++
				spawnWorker(workerID)
			}

		} else {
			// just wait it out; higher-ID workers will self-terminate by checking ActiveTarget in their main loop
			select {
			case <-ctx.Done():
				return
			case <-time.After(duration):
			}
		}

		// Update current VU count for the next stage
		currentVUs = targetVUs
	}
}
