package metrics

import (
	"sync"
	"testing"
)

func TestAtomicHistogram_SingleThread_Percentiles(t *testing.T) {
	h := NewAtomicHistogram()

	// Record values from 1ms to 100ms.
	for i := int64(1); i <= 100; i++ {
		h.Record(i)
	}

	if h.Min() != 1 {
		t.Errorf("Expected Min 1, got %d", h.Min())
	}
	if h.Max() != 100 {
		t.Errorf("Expected Max 100, got %d", h.Max())
	}

	// Verify p50, p90, p95, and p99.
	if p50 := h.Percentile(50); p50 != 50 {
		t.Errorf("Expected p50 to be 50, got %d", p50)
	}
	if p95 := h.Percentile(95); p95 != 95 {
		t.Errorf("Expected p95 to be 95, got %d", p95)
	}
	if p99 := h.Percentile(99); p99 != 99 {
		t.Errorf("Expected p99 to be 99, got %d", p99)
	}
}

func TestAtomicHistogram_Tier2_StepCalculation(t *testing.T) {
	h := NewAtomicHistogram()

	// Record a value beyond Tier 1 (> 10000ms).
	val := int64(15025)
	h.Record(val)

	if h.Max() != val {
		t.Errorf("Expected Max %d, got %d", val, h.Max())
	}

	// Tier 2 uses 10ms buckets, so the percentile is bucket-aligned.
	p := h.Percentile(100)
	if p < 15020 || p > 15030 {
		t.Errorf("Expected Tier 2 step-aligned percentile around 15020, got %d", p)
	}
}

func TestAtomicHistogram_ConcurrentSafety(t *testing.T) {
	h := NewAtomicHistogram()
	workers := 20
	iterations := 10000

	var wg sync.WaitGroup
	wg.Add(workers)

	for w := 0; w < workers; w++ {
		go func(workerId int) {
			defer wg.Done()
			for i := 1; i <= iterations; i++ {
				h.Record(int64((workerId * 10) + (i % 500)))
			}
		}(w)
	}

	wg.Wait()

	expectedTotal := int64(workers * iterations)
	if h.TotalCount != expectedTotal {
		t.Errorf("Expected TotalCount %d, got %d", expectedTotal, h.TotalCount)
	}
}
