package metrics

import (
	"sync"
	"sync/atomic"
)

type BranchMetrics struct {
	TrueCount  int64
	_          [7]uint64
	FalseCount int64
}

func (m *BranchMetrics) Record(isTrue bool) {
	if isTrue {
		atomic.AddInt64(&m.TrueCount, 1)
	} else {
		atomic.AddInt64(&m.FalseCount, 1)
	}
}

func (m *BranchMetrics) Snapshot() (trueCount, falseCount int64) {
	return atomic.LoadInt64(&m.TrueCount), atomic.LoadInt64(&m.FalseCount)
}

type LoopMetrics struct {
	TotalEntered    int64
	_               [7]uint64
	TotalIterations int64
}

func (m *LoopMetrics) Record(iterations int32) {
	atomic.AddInt64(&m.TotalEntered, 1)
	atomic.AddInt64(&m.TotalIterations, int64(iterations))
}

func (m *LoopMetrics) Snapshot() (entered, totalIters int64) {
	return atomic.LoadInt64(&m.TotalEntered), atomic.LoadInt64(&m.TotalIterations)
}

type PollMetrics struct {
	TotalEntered int64
	_            [7]uint64
	SuccessCount int64
	_            [7]uint64
	Exhausted    int64 // Retries exhausted without success
}

func (m *PollMetrics) Record(success bool) {
	atomic.AddInt64(&m.TotalEntered, 1)
	if success {
		atomic.AddInt64(&m.SuccessCount, 1)
	} else {
		atomic.AddInt64(&m.Exhausted, 1)
	}
}

func (m *PollMetrics) Snapshot() (entered, success, exhausted int64) {
	return atomic.LoadInt64(&m.TotalEntered), atomic.LoadInt64(&m.SuccessCount), atomic.LoadInt64(&m.Exhausted)
}

type MatchMetrics struct {
	Cases sync.Map // Key: case name; value: *int64
}

func (m *MatchMetrics) Record(caseName string) {
	if caseName == "" {
		caseName = "default"
	}

	if val, ok := m.Cases.Load(caseName); ok {
		atomic.AddInt64(val.(*int64), 1)
		return
	}

	val, _ := m.Cases.LoadOrStore(caseName, new(int64))
	atomic.AddInt64(val.(*int64), 1)
}

func (m *MatchMetrics) GetAll() map[string]int64 {
	res := make(map[string]int64)
	m.Cases.Range(func(key, value any) bool {
		res[key.(string)] = atomic.LoadInt64(value.(*int64))
		return true
	})
	return res
}

type ScriptMetrics struct {
	TotalCalls     int64
	_              [7]uint64
	TotalLatencyUs int64
	_              [7]uint64
	FailCount      int64
}

func (m *ScriptMetrics) Record(latencyUs int64, isFail bool) {
	atomic.AddInt64(&m.TotalCalls, 1)
	atomic.AddInt64(&m.TotalLatencyUs, latencyUs)
	if isFail {
		atomic.AddInt64(&m.FailCount, 1)
	}
}

func (m *ScriptMetrics) Snapshot() (calls, totalLatencyUs, failCount int64) {
	return atomic.LoadInt64(&m.TotalCalls), atomic.LoadInt64(&m.TotalLatencyUs), atomic.LoadInt64(&m.FailCount)
}
