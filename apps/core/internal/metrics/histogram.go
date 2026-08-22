package metrics

import (
	"sync/atomic"
)

// Calculates p90, p95, p99, Min, and Max with zero allocation
type AtomicHistogram struct {
	TotalCount int64
	MinVal     int64
	MaxVal     int64
	Buckets    [10000]int64 // 0ms -> 10,000ms
}

func NewAtomicHistogram() *AtomicHistogram {
	return &AtomicHistogram{
		MinVal: -1, // Uninitialized sentinel value
	}
}

func (h *AtomicHistogram) Record(val int64) {
	atomic.AddInt64(&h.TotalCount, 1)
	for {
		currMax := atomic.LoadInt64(&h.MaxVal)
		if val <= currMax {
			break
		}
		if atomic.CompareAndSwapInt64(&h.MaxVal, currMax, val) {
			break
		}
	}
	for {
		currMin := atomic.LoadInt64(&h.MinVal)
		if currMin != -1 && val >= currMin {
			break
		}
		if atomic.CompareAndSwapInt64(&h.MinVal, currMin, val) {
			break
		}
	}
	idx := val
	if idx < 0 {
		idx = 0
	} else if idx >= 10000 {
		idx = 9999
	}
	atomic.AddInt64(&h.Buckets[idx], 1)
}

func (h *AtomicHistogram) Percentile(p float64) int64 {
	total := atomic.LoadInt64(&h.TotalCount)
	if total == 0 {
		return 0
	}
	target := int64(float64(total) * (p / 100.0))
	var sum int64 = 0
	for i := 0; i < 10000; i++ {
		sum += atomic.LoadInt64(&h.Buckets[i])
		if sum >= target {
			return int64(i)
		}
	}
	return atomic.LoadInt64(&h.MaxVal)
}

func (h *AtomicHistogram) Min() int64 {
	m := atomic.LoadInt64(&h.MinVal)
	if m == -1 {
		return 0
	}
	return m
}

func (h *AtomicHistogram) Max() int64 {
	return atomic.LoadInt64(&h.MaxVal)
}
