package metrics

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestAggregator_DrainOnShutdown(t *testing.T) {
	agg := NewAggregator(1000)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go agg.Run(ctx, done)

	// Enqueue 100 events into the aggregator
	for i := 0; i < 100; i++ {
		agg.PushEvent(MetricEvent{
			WorkerID:    1,
			NodeID:      "node_test",
			ErrorMsg:    "Network Timeout",
			IsLogicFail: true,
		})
	}

	// Cancel context immediately to trigger graceful drain
	cancel()

	select {
	case <-done:
		// Aggregator must drain all 100 events before closing done channel
		_, _, _, _, logicFail, _, _, _ := agg.Metrics.Snapshot()
		if logicFail != 100 {
			t.Fatalf("premature shutdown: expected 100 processed events, got %d", logicFail)
		}
		errs := agg.Metrics.GetTopErrors()
		if errs["Network Timeout"] != 100 {
			t.Fatalf("error count mismatch: expected 100, got %d", errs["Network Timeout"])
		}
	case <-time.After(1 * time.Second):
		t.Fatal("aggregator failed to shut down within timeout")
	}
}

func TestAggregator_NonBlockingDrop_OnOverflow(t *testing.T) {
	// Initialize minimal buffer (capacity 2) without starting Run() consumer loop
	agg := NewAggregator(2)

	// Enqueue 5 consecutive events
	for i := 0; i < 5; i++ {
		agg.PushEvent(MetricEvent{
			WorkerID: i,
			ErrorMsg: "Fast flood",
		})
	}

	// 2 events remain buffered; remaining 3 must be dropped immediately
	dropped := atomic.LoadInt64(&agg.DroppedEvents)
	if dropped != 3 {
		t.Fatalf("dropped events count mismatch: expected 3, got %d", dropped)
	}
}
