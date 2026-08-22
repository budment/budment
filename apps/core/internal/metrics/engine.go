package metrics

import (
	"sync"
	"sync/atomic"
)

type TimeSeriesPoint struct {
	Timestamp int64   `json:"timestamp"`
	RPS       float64 `json:"rps"`
	VUs       int64   `json:"vus"`
}

type EngineMetrics struct {
	TotalIterations int64
	_               [7]uint64
	TotalRequests   int64
	_               [7]uint64
	SuccessCount    int64
	_               [7]uint64
	FailCount       int64
	_               [7]uint64
	TotalDataSent   int64
	_               [7]uint64
	TotalDataRecv   int64
	_               [7]uint64
	ActiveVUs       int64
	_               [7]uint64

	LogicFailCount    int64
	_                 [7]uint64
	IterationDuration *AtomicHistogram    // Per-iteration duration metrics
	Custom            *CustomMetricsStore // User-defined JS metrics

	nodes    sync.Map
	branches sync.Map
	loops    sync.Map
	polls    sync.Map
	matches  sync.Map
	scripts  sync.Map

	mu           sync.RWMutex
	ErrorCounts  map[string]int
	TimeSeries   []TimeSeriesPoint
	historySize  int
	historyHead  int
	historyCount int
	RPSHistory   []float64
	VUHistory    []int64
}

func NewEngineMetrics() *EngineMetrics {
	size := 60
	return &EngineMetrics{
		IterationDuration: NewAtomicHistogram(),
		Custom:            NewCustomMetricsStore(),
		ErrorCounts:       make(map[string]int),
		TimeSeries:        make([]TimeSeriesPoint, 0, 1000),
		historySize:       size,
		RPSHistory:        make([]float64, size),
		VUHistory:         make([]int64, size),
	}
}

func (m *EngineMetrics) GetNode(nodeID string) *NodeMetrics {
	if nodeID == "" {
		nodeID = "unknown"
	}
	val, ok := m.nodes.Load(nodeID)
	if !ok {
		newNode := NewNodeMetrics()
		val, _ = m.nodes.LoadOrStore(nodeID, newNode)
	}
	return val.(*NodeMetrics)
}

func (m *EngineMetrics) GetAllNodes() map[string]*NodeMetrics {
	result := make(map[string]*NodeMetrics)
	m.nodes.Range(func(key, value any) bool {
		result[key.(string)] = value.(*NodeMetrics)
		return true
	})
	return result
}

func (m *EngineMetrics) RecordIteration(durationMs int64) {
	atomic.AddInt64(&m.TotalIterations, 1)
	m.IterationDuration.Record(durationMs)
}

func (m *EngineMetrics) RecordRequest(nodeID string, success bool, latencyUs, ttfbUs, tcpUs, tlsUs int64, bytesSent, bytesRecv int64, statusCode int) {
	atomic.AddInt64(&m.TotalRequests, 1)
	atomic.AddInt64(&m.TotalDataSent, bytesSent)
	atomic.AddInt64(&m.TotalDataRecv, bytesRecv)

	if success {
		atomic.AddInt64(&m.SuccessCount, 1)
	} else {
		atomic.AddInt64(&m.FailCount, 1)
	}

	node := m.GetNode(nodeID)
	node.Record(success, latencyUs, ttfbUs, tcpUs, tlsUs, statusCode)
}

func (m *EngineMetrics) AddActiveVU(delta int64) {
	atomic.AddInt64(&m.ActiveVUs, delta)
}

func (m *EngineMetrics) RecordError(errMsg string) {
	if errMsg == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ErrorCounts[errMsg]++
}

func (m *EngineMetrics) AppendHistory(rps float64, vus int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.RPSHistory[m.historyHead] = rps
	m.VUHistory[m.historyHead] = vus
	m.historyHead = (m.historyHead + 1) % m.historySize
	if m.historyCount < m.historySize {
		m.historyCount++
	}
}

func (m *EngineMetrics) GetHistory() (rps []float64, vus []int64) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	rps = make([]float64, 0, m.historyCount)
	vus = make([]int64, 0, m.historyCount)
	for i := 0; i < m.historyCount; i++ {
		idx := (m.historyHead - m.historyCount + i + m.historySize) % m.historySize
		rps = append(rps, m.RPSHistory[idx])
		vus = append(vus, m.VUHistory[idx])
	}
	return rps, vus
}

func (m *EngineMetrics) Snapshot() (reqs, iters, success, netFail, logicFail, vus, dataSent, dataRecv int64) {
	reqs = atomic.LoadInt64(&m.TotalRequests)
	iters = atomic.LoadInt64(&m.TotalIterations)
	success = atomic.LoadInt64(&m.SuccessCount)
	netFail = atomic.LoadInt64(&m.FailCount)
	logicFail = atomic.LoadInt64(&m.LogicFailCount)
	vus = atomic.LoadInt64(&m.ActiveVUs)
	dataSent = atomic.LoadInt64(&m.TotalDataSent)
	dataRecv = atomic.LoadInt64(&m.TotalDataRecv)
	return
}

func (m *EngineMetrics) GetTopErrors() map[string]int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	errCopy := make(map[string]int)
	for k, v := range m.ErrorCounts {
		errCopy[k] = v
	}
	return errCopy
}

func (m *EngineMetrics) RecordLogicFail() {
	atomic.AddInt64(&m.LogicFailCount, 1)
}

func (m *EngineMetrics) RecordBranch(nodeID string, isTrue bool) {
	val, _ := m.branches.LoadOrStore(nodeID, &BranchMetrics{})
	val.(*BranchMetrics).Record(isTrue)
}

func (m *EngineMetrics) RecordLoop(nodeID string, iters int32) {
	val, _ := m.loops.LoadOrStore(nodeID, &LoopMetrics{})
	val.(*LoopMetrics).Record(iters)
}

func (m *EngineMetrics) RecordPoll(nodeID string, success bool) {
	val, _ := m.polls.LoadOrStore(nodeID, &PollMetrics{})
	val.(*PollMetrics).Record(success)
}

func (m *EngineMetrics) RecordMatch(nodeID string, caseName string) {
	val, _ := m.matches.LoadOrStore(nodeID, &MatchMetrics{})
	val.(*MatchMetrics).Record(caseName)
}

func (m *EngineMetrics) GetAllBranches() map[string]*BranchMetrics {
	res := make(map[string]*BranchMetrics)
	m.branches.Range(func(k, v any) bool { res[k.(string)] = v.(*BranchMetrics); return true })
	return res
}
func (m *EngineMetrics) GetAllLoops() map[string]*LoopMetrics {
	res := make(map[string]*LoopMetrics)
	m.loops.Range(func(k, v any) bool { res[k.(string)] = v.(*LoopMetrics); return true })
	return res
}
func (m *EngineMetrics) GetAllPolls() map[string]*PollMetrics {
	res := make(map[string]*PollMetrics)
	m.polls.Range(func(k, v any) bool { res[k.(string)] = v.(*PollMetrics); return true })
	return res
}
func (m *EngineMetrics) GetAllMatches() map[string]*MatchMetrics {
	res := make(map[string]*MatchMetrics)
	m.matches.Range(func(k, v any) bool { res[k.(string)] = v.(*MatchMetrics); return true })
	return res
}

func (m *EngineMetrics) RecordScript(nodeID string, latencyUs int64, isFail bool) {
	val, _ := m.scripts.LoadOrStore(nodeID, &ScriptMetrics{})
	val.(*ScriptMetrics).Record(latencyUs, isFail)
}

func (m *EngineMetrics) GetAllScripts() map[string]*ScriptMetrics {
	res := make(map[string]*ScriptMetrics)
	m.scripts.Range(func(k, v any) bool { res[k.(string)] = v.(*ScriptMetrics); return true })
	return res
}
