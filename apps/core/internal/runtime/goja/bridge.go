package goja

import "github.com/vunas/blaster/internal/runtime"

// Exposes metrics APIs
type JSMetricsAPI struct {
	bridge *JSBridge
}

func (m *JSMetricsAPI) Trend(name string, val float64) {
	if m.bridge.Sink != nil {
		m.bridge.Sink.RecordCustom(m.bridge.VuId, "trend", name, val)
	}
}
func (m *JSMetricsAPI) Counter(name string, val float64) {
	if m.bridge.Sink != nil {
		m.bridge.Sink.RecordCustom(m.bridge.VuId, "counter", name, val)
	}
}
func (m *JSMetricsAPI) Gauge(name string, val float64) {
	if m.bridge.Sink != nil {
		m.bridge.Sink.RecordCustom(m.bridge.VuId, "gauge", name, val)
	}
}

type JSBridge struct {
	WorkerScope                     runtime.Scope
	Local                           runtime.SharedState
	Global                          runtime.SharedState
	Sink                            runtime.MetricsSink
	Metrics                         *JSMetricsAPI
	VuId                            int    `json:"vuId"`
	Iteration                       int    `json:"iteration"`
	Scenario                        string `json:"scenario"`
	CurrentHook                     string
	AbortFlag, SkipFlag, RetryFlag  bool
	RetryScope                      string
	RetryDelay, RetryMax, SleepTime int
	BarrierName                     string
	BarrierOptions                  map[string]any
}

func NewJSBridge(global runtime.SharedState, local runtime.SharedState, sink runtime.MetricsSink) *JSBridge {
	b := &JSBridge{Global: global, Local: local, Sink: sink}
	b.Metrics = &JSMetricsAPI{bridge: b}
	return b
}

func (b *JSBridge) AttachWorker(scope runtime.Scope, local runtime.SharedState, vuId int, iteration int, scenario string) {
	b.WorkerScope = scope
	b.Local = local
	b.VuId = vuId
	b.Iteration = iteration
	b.Scenario = scenario
	b.CurrentHook = "unknown"

	b.AbortFlag, b.SkipFlag, b.RetryFlag = false, false, false
	b.RetryScope, b.RetryDelay, b.RetryMax, b.SleepTime = "hook", 0, 3, 0
	b.BarrierName, b.BarrierOptions = "", nil
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
		b.Sink.Log(b.VuId, b.CurrentHook, "INFO", msg)
	}
}

func (b *JSBridge) Warn(msg string) {
	if b.Sink != nil {
		b.Sink.Log(b.VuId, b.CurrentHook, "WARN", msg)
	}
}

func (b *JSBridge) Error(msg string) {
	if b.Sink != nil {
		b.Sink.Log(b.VuId, b.CurrentHook, "ERROR", msg)
	}
}

func (b *JSBridge) Tag(key string, value string) {
	if b.Sink != nil {
		b.Sink.Tag(b.VuId, key, value)
	}
}

func (b *JSBridge) Fail(reason string) {
	if b.Sink != nil {
		b.Sink.RecordEvent(b.VuId, b.CurrentHook, "FAIL", reason)
	}
}

func (b *JSBridge) Skip(reason string) {
	b.SkipFlag = true
	panic("BLASTER_SKIP")
}

func (b *JSBridge) Abort(reason string) {
	b.AbortFlag = true
	if b.Sink != nil {
		b.Sink.RecordEvent(b.VuId, b.CurrentHook, "ABORT", reason)
	}
	panic("BLASTER_ABORT")
}

func (b *JSBridge) Sleep(ms int) {
	b.SleepTime = ms
	panic("BLASTER_SLEEP")
}

func (b *JSBridge) Barrier(name string, options map[string]any) {
	b.BarrierName = name
	b.BarrierOptions = options
	panic("BLASTER_BARRIER")
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
		b.Local.StoreDistribution(key, items, fallback)
	}
}

func (b *JSBridge) DistributeRandom(key string, items []any) {
	if b.Local != nil {
		b.Local.StoreDistribution(key, items, nil)
	}
}
