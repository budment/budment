package engine

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestBarrierManager_QuorumRelease(t *testing.T) {
	bm := NewBarrierManager()
	quorum := 5
	barrierName := "start-gate"

	var unblockedCount int64
	var wg sync.WaitGroup
	ctx := context.Background()

	// Spawn goroutines to meet quorum
	for i := 0; i < quorum; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bm.ArriveAndWait(ctx, barrierName, quorum, 2*time.Second)
			atomic.AddInt64(&unblockedCount, 1)
		}()
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		if count := atomic.LoadInt64(&unblockedCount); count != int64(quorum) {
			t.Fatalf("unblocked goroutines mismatch: expected %d, got %d", quorum, count)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("deadlock: goroutines failed to pass barrier")
	}
}

func TestBarrierManager_ContextCancellation(t *testing.T) {
	bm := NewBarrierManager()
	quorum := 10 // Quorum is 10, but only 2 goroutines arrive
	barrierName := "unmet-gate"

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bm.ArriveAndWait(ctx, barrierName, quorum, 5*time.Second)
		}()
	}

	time.AfterFunc(50*time.Millisecond, cancel)

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Workers exited promptly upon cancellation without goroutine leak
	case <-time.After(1 * time.Second):
		t.Fatal("workers timed out instead of exiting upon context cancellation")
	}
}
