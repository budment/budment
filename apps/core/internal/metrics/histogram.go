package metrics

import (
	"math"
	"sync/atomic"
)

const (
	tier1Max     = 10000
	tier2Max     = 60000
	tier2Step    = 10
	totalBuckets = tier1Max + (tier2Max-tier1Max)/tier2Step
)

// Calculates p90, p95, p99, Min, and Max with zero allocation
type AtomicHistogram struct {
	TotalCount int64
	MinVal     int64
	MaxVal     int64
	Buckets    [totalBuckets]int64
}

func NewAtomicHistogram() *AtomicHistogram {
	return &AtomicHistogram{
		MinVal: -1, // Uninitialized sentinel value
	}
}

func (h *AtomicHistogram) Record(val int64) {
	if val < 0 {
		val = 0
	}

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
	var idx int
	if val < tier1Max {
		idx = int(val)
	} else if val < tier2Max {
		idx = tier1Max + int((val-tier1Max)/tier2Step)
	} else {
		idx = totalBuckets - 1
	}
	atomic.AddInt64(&h.Buckets[idx], 1)
}

func (h *AtomicHistogram) Percentile(p float64) int64 {
	total := atomic.LoadInt64(&h.TotalCount)
	if total == 0 {
		return 0
	}

	if p <= 0 {
		return h.Min()
	}
	if p >= 100 {
		return h.Max()
	}

	target := int64(math.Ceil(float64(total) * (p / 100.0)))
	if target < 1 {
		target = 1
	}

	var sum int64 = 0
	for i := 0; i < totalBuckets; i++ {
		sum += atomic.LoadInt64(&h.Buckets[i])
		if sum >= target {
			if i < tier1Max {
				return int64(i)
			}
			if i == totalBuckets-1 && atomic.LoadInt64(&h.MaxVal) >= tier2Max {
				return atomic.LoadInt64(&h.MaxVal)
			}
			return int64(tier1Max + (i-tier1Max)*tier2Step)
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
