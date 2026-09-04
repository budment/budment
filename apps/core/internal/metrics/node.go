package metrics

import (
	"sync/atomic"
)

// Stores isolated metrics per API/Logic node
type NodeMetrics struct {
	Name   string
	Method string

	TotalRequests  int64
	_              [7]uint64
	SuccessCount   int64
	_              [7]uint64
	FailCount      int64
	_              [7]uint64
	TotalLatencyUs int64
	_              [7]uint64

	StatusCodes [600]int64
	ReqDuration *AtomicHistogram
	TTFB        *AtomicHistogram
	TCPConn     *AtomicHistogram
	TLSHand     *AtomicHistogram
}

func NewNodeMetrics() *NodeMetrics {
	return &NodeMetrics{
		ReqDuration: NewAtomicHistogram(),
		TTFB:        NewAtomicHistogram(),
		TCPConn:     NewAtomicHistogram(),
		TLSHand:     NewAtomicHistogram(),
	}
}

func (n *NodeMetrics) Record(success bool, latencyUs, ttfbUs, tcpUs, tlsUs int64, statusCode int) {
	atomic.AddInt64(&n.TotalRequests, 1)
	atomic.AddInt64(&n.TotalLatencyUs, latencyUs)
	if success {
		atomic.AddInt64(&n.SuccessCount, 1)
	} else {
		atomic.AddInt64(&n.FailCount, 1)
	}

	if statusCode >= 0 && statusCode < 600 {
		atomic.AddInt64(&n.StatusCodes[statusCode], 1)
	}

	n.ReqDuration.Record(latencyUs / 1000)
	n.TTFB.Record(ttfbUs / 1000)
	n.TCPConn.Record(tcpUs / 1000)
	n.TLSHand.Record(tlsUs / 1000)
}

func (n *NodeMetrics) Snapshot() (reqs, success, fail, totalLatencyUs int64) {
	reqs = atomic.LoadInt64(&n.TotalRequests)
	success = atomic.LoadInt64(&n.SuccessCount)
	fail = atomic.LoadInt64(&n.FailCount)
	totalLatencyUs = atomic.LoadInt64(&n.TotalLatencyUs)
	return
}

func (n *NodeMetrics) GetStatusCodes() map[int]int64 {
	codes := make(map[int]int64)
	for i := 0; i < 600; i++ {
		c := atomic.LoadInt64(&n.StatusCodes[i])
		if c > 0 {
			codes[i] = c
		}
	}
	return codes
}
