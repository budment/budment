package goja

import (
	"fmt"
)

type Scope interface {
	Set(key string, val any)
	Get(key string) (any, bool)
	Delete(key string)
}

type SharedState interface {
	Set(key string, val any)
	Get(key string) (any, bool)
	Push(queueName string, val any) bool
	Pop(queueName string) any
	StoreDistribution(key string, items []any)
}

type MetricsSink interface {
	Log(workerID int, level string, msg string)
	RecordEvent(workerID int, eventType string, reason string)
	Tag(workerID int, key string, value string)
	RecordCustom(workerID int, mType string, name string, val float64)
}

// Exposes metrics APIs
type JSMetricsAPI struct {
	bridge *JSBridge
}

func (m *JSMetricsAPI) Trend(name string, val float64) {
	m.bridge.Sink.RecordCustom(m.bridge.VuId, "trend", name, val)
}
func (m *JSMetricsAPI) Counter(name string, val float64) {
	m.bridge.Sink.RecordCustom(m.bridge.VuId, "counter", name, val)
}
func (m *JSMetricsAPI) Gauge(name string, val float64) {
	m.bridge.Sink.RecordCustom(m.bridge.VuId, "gauge", name, val)
}

type JSBridge struct {
	WorkerScope                     Scope
	Local                           SharedState
	Global                          SharedState
	Sink                            MetricsSink
	Metrics                         *JSMetricsAPI
	VuId                            int    `json:"vuId"`
	Iteration                       int    `json:"iteration"`
	Scenario                        string `json:"scenario"`
	CurrentHook                     string
	AbortFlag, SkipFlag, RetryFlag  bool
	RetryScope                      string
	RetryDelay, RetryMax, SleepTime int
	SyncName                        string
	SyncOptions                     map[string]any
}

func NewJSBridge(global SharedState, local SharedState, sink MetricsSink) *JSBridge {
	b := &JSBridge{Global: global, Local: local, Sink: sink}
	b.Metrics = &JSMetricsAPI{bridge: b}
	return b
}

func (b *JSBridge) AttachWorker(scope Scope, local SharedState, vuId int, iteration int, scenario string) {
	b.WorkerScope = scope
	b.Local = local
	b.VuId = vuId
	b.Iteration = iteration
	b.Scenario = scenario
	b.CurrentHook = "unknown"

	b.AbortFlag, b.SkipFlag, b.RetryFlag = false, false, false
	b.RetryScope, b.RetryDelay, b.RetryMax, b.SleepTime = "hook", 0, 3, 0
	b.SyncName, b.SyncOptions = "", nil
}

func (b *JSBridge) Set(key string, val any) {
	if b.WorkerScope != nil {
		b.WorkerScope.Set(key, val)
	}
}

func (b *JSBridge) Get(key string) any {
	if b.WorkerScope != nil {
		v, ok := b.WorkerScope.Get(key)
		if ok {
			return v
		}
	}
	return nil
}

func (b *JSBridge) Delete(key string) {
	if b.WorkerScope != nil {
		b.WorkerScope.Delete(key)
	}
}

func (b *JSBridge) Log(msg string) {
	if b.Sink != nil {
		hookCtx := ""
		if b.CurrentHook != "unknown" && b.CurrentHook != "" {
			hookCtx = "[Hook: " + b.CurrentHook + "] "
		}
		b.Sink.Log(b.VuId, "INFO", hookCtx+msg)
	}
}

func (b *JSBridge) Warn(msg string) {
	if b.Sink != nil {
		hookCtx := ""
		if b.CurrentHook != "unknown" && b.CurrentHook != "" {
			hookCtx = "[Hook: " + b.CurrentHook + "] "
		}
		b.Sink.Log(b.VuId, "WARN", hookCtx+msg)
	}
}

func (b *JSBridge) Error(msg string) {
	if b.Sink != nil {
		hookCtx := ""
		if b.CurrentHook != "unknown" && b.CurrentHook != "" {
			hookCtx = "[Hook: " + b.CurrentHook + "] "
		}
		b.Sink.Log(b.VuId, "ERROR", hookCtx+msg)
	}
}

func (b *JSBridge) Tag(key string, value string) {
	if b.Sink != nil {
		b.Sink.Tag(b.VuId, key, value)
	}
}

func (b *JSBridge) Fail(reason string) {
	if b.Sink != nil {
		b.Sink.RecordEvent(b.VuId, "FAIL", fmt.Sprintf("[Hook: %s] %s", b.CurrentHook, reason))
	}
}

func (b *JSBridge) Skip(reason string) {
	b.SkipFlag = true
	panic("BLASTER_SKIP")
}

func (b *JSBridge) Abort(reason string) {
	b.AbortFlag = true
	if b.Sink != nil {
		b.Sink.RecordEvent(b.VuId, "ABORT", fmt.Sprintf("[Hook: %s] %s", b.CurrentHook, reason))
	}
	panic("BLASTER_ABORT")
}

func (b *JSBridge) Sleep(ms int) {
	b.SleepTime = ms
	panic("BLASTER_SLEEP")
}

func (b *JSBridge) Sync(name string, options map[string]any) {
	b.SyncName = name
	b.SyncOptions = options
	panic("BLASTER_SYNC")
}

func (b *JSBridge) Retry(options map[string]any) {
	b.RetryFlag = true
	if options != nil {
		if scope, ok := options["scope"].(string); ok {
			b.RetryScope = scope
		}
		if delay, ok := options["delay"].(float64); ok {
			b.RetryDelay = int(delay)
		}
		if max, ok := options["maxAttempts"].(float64); ok {
			b.RetryMax = int(max)
		}
	}
	panic("BLASTER_RETRY")
}

func (b *JSBridge) Distribute(key string, items []any, fallback any) {
	if b.Local != nil {
		b.Local.StoreDistribution(key, items)
	}
}

func (b *JSBridge) DistributeRandom(key string, items []any) {
	if b.Local != nil {
		b.Local.StoreDistribution(key, items)
	}
}
