package metrics

import (
	"sync/atomic"
)

// Stores isolated metrics per API/Logic node
type NodeMetrics struct {
	Name           string
	TotalRequests  int64
	SuccessCount   int64
	FailCount      int64
	TotalLatencyUs int64
	StatusCodes    [600]int64
	ReqDuration    *AtomicHistogram
	TTFB           *AtomicHistogram
	TCPConn        *AtomicHistogram
	TLSHand        *AtomicHistogram
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
