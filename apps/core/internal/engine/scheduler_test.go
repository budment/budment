package engine

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vunas/blaster/internal/config"
)

func TestConstantVUScheduler_Spawn(t *testing.T) {
	vus := 10
	scheduler := &ConstantVUScheduler{VUs: vus}

	var activeTarget int64
	var spawnedSlots []int
	var mu sync.Mutex

	spawnFunc := func(slot int, id int) {
		mu.Lock()
		spawnedSlots = append(spawnedSlots, slot)
		mu.Unlock()
	}

	scheduler.Start(context.Background(), spawnFunc, &activeTarget)

	if target := atomic.LoadInt64(&activeTarget); target != int64(vus) {
		t.Errorf("activeTarget mismatch: expected %d, got %d", vus, target)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(spawnedSlots) != vus {
		t.Fatalf("spawned workers mismatch: expected %d, got %d", vus, len(spawnedSlots))
	}
	for i, slot := range spawnedSlots {
		if slot != i+1 {
			t.Errorf("slot order mismatch at index %d: expected %d, got %d", i, i+1, slot)
		}
	}
}

func TestRampingScheduler_EarlyCancellation(t *testing.T) {
	// 5s target stage cancelled after 100ms
	stages := []config.Stage{
		{Duration: "5s", Target: 50},
	}
	scheduler := NewRampingScheduler(stages)

	ctx, cancel := context.WithCancel(context.Background())
	var activeTarget int64
	var spawnCount int64

	spawnFunc := func(slot int, id int) {
		atomic.AddInt64(&spawnCount, 1)
	}

	done := make(chan struct{})
	go func() {
		scheduler.Start(ctx, spawnFunc, &activeTarget)
		close(done)
	}()

	// Trigger early cancellation
	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// Scheduler cleanly aborted ticker loop upon context cancellation
	case <-time.After(1 * time.Second):
		t.Fatal("RampingScheduler failed to stop after context cancellation")
	}
}
