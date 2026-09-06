package runner

import (
	"context"

	"github.com/vunas/blaster/internal/template"
)

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
	ApplyMutation(meta map[string]template.Expression, payload template.Expression, scope template.ScopeProvider)
}

type ProtocolResponse interface {
	Json(selector ...string) any
	Contains(substr string) bool
	Assert(expectCode int, expectContains string, extractRules map[string]string, returnCode int) (isFailed bool, extracted map[string]any)
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
