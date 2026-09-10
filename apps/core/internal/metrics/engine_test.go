package metrics

import (
	"fmt"
	"sync"
	"testing"
)

func TestEngineMetrics_ConcurrentRecording_Integrity(t *testing.T) {
	m := NewEngineMetrics(nil)
	workers := 10
	iterationsPerWorker := 100

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(wID int) {
			defer wg.Done()
			for it := 0; it < iterationsPerWorker; it++ {
				// Record single iteration
				m.RecordIteration(15)

				// Record 1 successful and 1 failed request
				m.RecordRequest("node_get", "GET", "FetchUser", true, 5000, 2000, 1000, 1000, 128, 512, 200)
				m.RecordRequest("node_post", "POST", "CreateUser", false, 10000, 4000, 1000, 1000, 256, 128, 500)

				m.RecordLogicFail()
			}
		}(i)
	}
	wg.Wait()

	reqs, iters, success, netFail, logicFail, _, dataSent, dataRecv := m.Snapshot()
	expectedIters := int64(workers * iterationsPerWorker)
	expectedReqs := expectedIters * 2

	if iters != expectedIters {
		t.Fatalf("total iterations mismatch: expected %d, got %d", expectedIters, iters)
	}
	if reqs != expectedReqs {
		t.Fatalf("total requests mismatch: expected %d, got %d", expectedReqs, reqs)
	}
	if success != expectedIters {
		t.Fatalf("success count mismatch: expected %d, got %d", expectedIters, success)
	}
	if netFail != expectedIters {
		t.Fatalf("failure count mismatch: expected %d, got %d", expectedIters, netFail)
	}
	if logicFail != expectedIters {
		t.Fatalf("logic failure count mismatch: expected %d, got %d", expectedIters, logicFail)
	}
	if dataSent <= 0 || dataRecv <= 0 {
		t.Fatalf("data transfer metrics not accumulated properly")
	}
}

func TestEngineMetrics_ErrorCounts_CappingLimit(t *testing.T) {
	m := NewEngineMetrics(nil)

	// Record 250 distinct errors exceeding maxDistinctErrors threshold (200)
	for i := 0; i < 250; i++ {
		m.RecordError(fmt.Sprintf("HTTP 500: Error code %d", i))
	}

	errMap := m.GetTopErrors()

	// Map size must not exceed maxDistinctErrors + 1 (allocated for "[Other errors]")
	if len(errMap) > maxDistinctErrors+1 {
		t.Fatalf("error map size (%d) exceeded max limit (%d)", len(errMap), maxDistinctErrors)
	}

	if otherCount, exists := errMap["[Other errors]"]; !exists || otherCount != 50 {
		t.Fatalf("expected 50 overflowed errors under '[Other errors]', got %d (exists: %v)", otherCount, exists)
	}
}

func TestEngineMetrics_CircularBuffer_RPSHistory(t *testing.T) {
	m := NewEngineMetrics(nil)

	// Record 75 ticks exceeding default buffer size (60)
	for i := 1; i <= 75; i++ {
		m.RecordTick(float64(i), int64(i*2))
	}

	rps, vus := m.GetHistory()

	// Retain at most the 60 most recent entries
	if len(rps) != 60 || len(vus) != 60 {
		t.Fatalf("history length mismatch: expected 60, got rps=%d, vus=%d", len(rps), len(vus))
	}

	// Oldest entry should be tick 16 (75 - 60 + 1)
	if rps[0] != 16.0 || vus[0] != 32 {
		t.Fatalf("circular buffer index error: oldest tick expected rps=16.0 vus=32, got rps=%f vus=%d", rps[0], vus[0])
	}

	// Newest entry should be tick 75
	if rps[59] != 75.0 || vus[59] != 150 {
		t.Fatalf("circular buffer index error: newest tick expected rps=75.0 vus=150, got rps=%f vus=%d", rps[59], vus[59])
	}
}
