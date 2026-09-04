package runner

import "context"

type ExecutionResult struct {
	IsSuccess    bool
	LatencyUs    int64
	TTFBUs       int64
	TCPConnUs    int64
	TLSHandUs    int64
	BytesOut     int64
	BytesIn      int64
	Code         int
	ErrorMessage string
}

type ProtocolRequest interface {
	Set(body any, options map[string]any)
	Json(selector ...string) any
}

type ProtocolResponse interface {
	Json(selector ...string) any
	Contains(substr string) bool
	Release()
}

type ProtocolRunner interface {
	AcquireRequest(method, target string) ProtocolRequest
	ReleaseRequest(req ProtocolRequest)
	Execute(ctx context.Context, req ProtocolRequest) (ProtocolResponse, ExecutionResult)
}

type Manager struct {
	runners map[string]ProtocolRunner
}

func NewManager() *Manager {
	return &Manager{runners: make(map[string]ProtocolRunner)}
}

func (m *Manager) Register(protocol string, r ProtocolRunner) {
	m.runners[protocol] = r
}

func (m *Manager) Get(protocol string) ProtocolRunner {
	return m.runners[protocol]
}
