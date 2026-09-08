package goja

import (
	"math"
	"sync"
	"testing"

	"github.com/dop251/goja"
)

type stubVUContext struct {
	data      map[string]any
	mu        sync.RWMutex
	abortFlag bool
}

func newStubVUContext() *stubVUContext {
	return &stubVUContext{data: make(map[string]any)}
}

func (s *stubVUContext) Get(key string) (any, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.data[key]
	return v, ok
}

func (s *stubVUContext) Set(key string, val any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = val
}

func (s *stubVUContext) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
}

func (s *stubVUContext) SetFlags(abort bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.abortFlag = abort
}

func (s *stubVUContext) IsAbortFlag() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.abortFlag
}

type stubSink struct {
	logs   []string
	events []string
	custom []string
	mu     sync.Mutex
}

func (s *stubSink) Log(vuId int, nodeID, level, msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logs = append(s.logs, level+":"+msg)
}

func (s *stubSink) Tag(vuId int, k, v string) {}

func (s *stubSink) RecordEvent(vuId int, nodeID, eventType, reason string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, eventType+":"+reason)
}

func (s *stubSink) RecordCustom(vuId int, mType, name string, val float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.custom = append(s.custom, mType+":"+name)
}

func TestJSBridge_ColdStartGuards(t *testing.T) {
	sink := &stubSink{}
	bridge := NewJSBridge(nil, sink)

	// In cold-start initialization (IsRuntime = false):
	// Operations must be no-ops without panics or state mutation
	bridge.Set("foo", "bar")
	if val := bridge.Get("foo"); val != nil {
		t.Fatalf("expected nil on cold-start Get, got %v", val)
	}

	// Must not trigger BLASTER_SLEEP or BLASTER_ABORT panics when IsRuntime = false
	bridge.Sleep(1.0)
	bridge.Abort("stop")
	bridge.Barrier("sync", nil)

	if bridge.SleepTime != 0 || bridge.AbortFlag {
		t.Fatal("bridge state mutated while IsRuntime is false")
	}
}

func TestJSBridge_AttachAndWorkerScope(t *testing.T) {
	sink := &stubSink{}
	bridge := NewJSBridge(nil, sink)
	scope := newStubVUContext()

	// Attach worker context to bridge
	bridge.AttachWorker(scope, nil, 10, 1, "ScenarioA")
	if !bridge.IsRuntime || bridge.VuId != 10 {
		t.Fatalf("AttachWorker failed to update bridge state")
	}

	bridge.Set("user_id", 42)
	if val, ok := scope.Get("user_id"); !ok || val != 42 {
		t.Fatalf("scope failed to receive value from bridge.Set: expected 42, got %v", val)
	}

	if val := bridge.Get("user_id"); val != 42 {
		t.Fatalf("bridge.Get failed to retrieve value from scope: expected 42, got %v", val)
	}

	// Detach worker on pool release
	bridge.AttachWorker(nil, nil, 0, 0, "")
	if bridge.IsRuntime {
		t.Fatal("expected IsRuntime to be false after detaching worker")
	}
}

func TestJSBridge_ControlFlowPanics(t *testing.T) {
	sink := &stubSink{}
	bridge := NewJSBridge(nil, sink)
	bridge.AttachWorker(newStubVUContext(), nil, 1, 0, "Test")

	// Sleep panic
	assertPanic(t, "BLASTER_SLEEP", func() {
		bridge.Sleep(2.5)
	})
	if bridge.SleepTime != 2500 {
		t.Fatalf("sleep time mismatch: expected 2500ms, got %d", bridge.SleepTime)
	}

	// Abort panic
	assertPanic(t, "BLASTER_ABORT", func() {
		bridge.Abort("assertion failed")
	})
	if !bridge.AbortFlag {
		t.Fatal("expected AbortFlag to be true")
	}

	// Barrier panic
	assertPanic(t, "BLASTER_BARRIER", func() {
		bridge.Barrier("gate_1", map[string]any{"quorum": 5.0})
	})
	if bridge.BarrierName != "gate_1" {
		t.Fatalf("barrier name mismatch: expected 'gate_1', got '%s'", bridge.BarrierName)
	}
}

func TestJSBridge_Metrics_SafeFloatSanitization(t *testing.T) {
	sink := &stubSink{}
	bridge := NewJSBridge(nil, sink)
	bridge.AttachWorker(newStubVUContext(), nil, 1, 0, "Test")
	vm := goja.New()

	// Valid numbers
	bridge.Metrics.Trend("latency", vm.ToValue(12.5))
	bridge.Metrics.Counter("requests", vm.ToValue(1))
	bridge.Metrics.Gauge("active_users", vm.ToValue(5))

	// Invalid/unsafe JS values (NaN, Infinity, undefined) must be safely dropped
	bridge.Metrics.Trend("invalid_nan", vm.ToValue(math.NaN()))
	bridge.Metrics.Trend("invalid_inf", vm.ToValue(math.Inf(1)))
	bridge.Metrics.Trend("invalid_undef", goja.Undefined())

	if len(sink.custom) != 3 {
		t.Fatalf("recorded metrics count mismatch: expected 3, got %d (%v)", len(sink.custom), sink.custom)
	}
}

func assertPanic(t *testing.T, expectedPrefix string, fn func()) {
	t.Helper()
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("expected panic containing '%s', got none", expectedPrefix)
		}
		rStr, ok := r.(string)
		if !ok || rStr != expectedPrefix {
			t.Fatalf("panic mismatch: expected '%s', got '%v'", expectedPrefix, r)
		}
	}()
	fn()
}
