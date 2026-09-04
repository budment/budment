package engine

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/vunas/blaster/internal/config"
)

type Scheduler interface {
	Start(ctx context.Context, spawnWorker func(slot int, id int), activeTarget *int32)
}

type ConstantVUScheduler struct{ VUs int }

func (s *ConstantVUScheduler) Start(ctx context.Context, spawnWorker func(slot int, id int), activeTarget *int32) {
	atomic.StoreInt32(activeTarget, int32(s.VUs))
	for slot := 1; slot <= s.VUs; slot++ {
		spawnWorker(slot, slot)
	}
}

type RampingScheduler struct {
	Stages []config.Stage
}

func NewRampingScheduler(stages []config.Stage) *RampingScheduler {
	return &RampingScheduler{Stages: stages}
}

func (s *RampingScheduler) Start(ctx context.Context, spawnWorker func(slot int, id int), activeTarget *int32) {
	currentVUs := 0
	totalSpawnedCount := 0

	for _, stage := range s.Stages {
		duration, err := time.ParseDuration(stage.Duration)
		if err != nil {
			continue
		}

		targetVUs := stage.Target
		tickDuration := 100 * time.Millisecond
		totalTicks := int(duration / tickDuration)
		if totalTicks <= 0 {
			totalTicks = 1
		}

		ticker := time.NewTicker(tickDuration)

		if targetVUs > currentVUs {
			diff := targetVUs - currentVUs
			vusPerTick := float64(diff) / float64(totalTicks)
			var accumulated float64
			spawnedSlots := currentVUs

			for i := 0; i < totalTicks; i++ {
				select {
				case <-ctx.Done():
					ticker.Stop()
					return
				case <-ticker.C:
					accumulated += vusPerTick
					targetForNow := currentVUs + int(accumulated)
					if targetForNow > targetVUs {
						targetForNow = targetVUs
					}
					atomic.StoreInt32(activeTarget, int32(targetForNow))

					for spawnedSlots < targetForNow {
						spawnedSlots++
						totalSpawnedCount++
						spawnWorker(spawnedSlots, totalSpawnedCount)
					}
				}
			}
			ticker.Stop()
			atomic.StoreInt32(activeTarget, int32(targetVUs))
			for spawnedSlots < targetVUs {
				spawnedSlots++
				totalSpawnedCount++
				spawnWorker(spawnedSlots, totalSpawnedCount)
			}

		} else if targetVUs < currentVUs {
			diff := currentVUs - targetVUs
			vusDropPerTick := float64(diff) / float64(totalTicks)
			var droppedAccum float64

			for i := 0; i < totalTicks; i++ {
				select {
				case <-ctx.Done():
					ticker.Stop()
					return
				case <-ticker.C:
					droppedAccum += vusDropPerTick
					targetForNow := currentVUs - int(droppedAccum)
					if targetForNow < targetVUs {
						targetForNow = targetVUs
					}
					atomic.StoreInt32(activeTarget, int32(targetForNow))
				}
			}
			ticker.Stop()
			atomic.StoreInt32(activeTarget, int32(targetVUs))

		} else {
			atomic.StoreInt32(activeTarget, int32(targetVUs))
			ticker.Stop()

			timer := time.NewTimer(duration)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
		}

		// Update current VU count for the next stage
		currentVUs = targetVUs
	}
}
