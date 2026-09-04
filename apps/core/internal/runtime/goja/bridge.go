package goja

import (
	"math"

	"github.com/dop251/goja"
	"github.com/vunas/blaster/internal/runtime"
)

// Exposes metrics APIs
type JSMetricsAPI struct{ bridge *JSBridge }

func getSafeFloat(val goja.Value) (float64, bool) {
	if val == nil || goja.IsUndefined(val) || goja.IsNull(val) {
		return 0, false
	}
	f := val.ToFloat()
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, false
	}
	return f, true
}

func (m *JSMetricsAPI) Trend(name string, val goja.Value) {
	if !m.bridge.IsRuntime || m.bridge.Sink == nil {
		return
	}
	if f, ok := getSafeFloat(val); ok {
		m.bridge.Sink.RecordCustom(m.bridge.VuId, "trend", name, f)
	}
}

func (m *JSMetricsAPI) Counter(name string, val goja.Value) {
	if !m.bridge.IsRuntime || m.bridge.Sink == nil {
		return
	}
	if f, ok := getSafeFloat(val); ok {
		m.bridge.Sink.RecordCustom(m.bridge.VuId, "counter", name, f)
	}
}

func (m *JSMetricsAPI) Gauge(name string, val goja.Value) {
	if !m.bridge.IsRuntime || m.bridge.Sink == nil {
		return
	}
	if f, ok := getSafeFloat(val); ok {
		m.bridge.Sink.RecordCustom(m.bridge.VuId, "gauge", name, f)
	}
}

type JSLocalAPI struct{ bridge *JSBridge }

func (l *JSLocalAPI) Get(k string) any {
	if !l.bridge.IsRuntime || l.bridge.Local == nil {
		return nil
	}
	v, _ := l.bridge.Local.Get(k)
	return v
}

func (l *JSLocalAPI) Set(k string, v any) {
	if !l.bridge.IsRuntime || l.bridge.Local == nil {
		return
	}
	l.bridge.Local.Set(k, v)
}

func (l *JSLocalAPI) Push(k string, v any) {
	if !l.bridge.IsRuntime || l.bridge.Local == nil {
		return
	}
	l.bridge.Local.Push(k, v)
}

func (l *JSLocalAPI) Pop(k string) any {
	if !l.bridge.IsRuntime || l.bridge.Local == nil {
		return nil
	}
	return l.bridge.Local.Pop(k)
}

type JSGlobalAPI struct{ bridge *JSBridge }

func (g *JSGlobalAPI) Get(k string) any {
	if !g.bridge.IsRuntime || g.bridge.Global == nil {
		return nil
	}
	v, _ := g.bridge.Global.Get(k)
	return v
}

func (g *JSGlobalAPI) Set(k string, v any) {
	if !g.bridge.IsRuntime || g.bridge.Global == nil {
		return
	}
	g.bridge.Global.Set(k, v)
}

func (g *JSGlobalAPI) Push(k string, v any) {
	if !g.bridge.IsRuntime || g.bridge.Global == nil {
		return
	}
	g.bridge.Global.Push(k, v)
}

func (g *JSGlobalAPI) Pop(k string) any {
	if !g.bridge.IsRuntime || g.bridge.Global == nil {
		return nil
	}
	return g.bridge.Global.Pop(k)
}

type JSBridge struct {
	WorkerScope    runtime.VUContext
	Local          runtime.SharedState
	Global         runtime.SharedState
	Sink           runtime.MetricsSink
	Metrics        *JSMetricsAPI
	LocalAPI       *JSLocalAPI
	GlobalAPI      *JSGlobalAPI
	IsRuntime      bool
	VuId           int    `json:"vuId"`
	Iteration      int    `json:"iteration"`
	Scenario       string `json:"scenario"`
	CurrentHook    string
	AbortFlag      bool
	SleepTime      int
	BarrierName    string
	BarrierOptions map[string]any
}

func NewJSBridge(global runtime.SharedState, sink runtime.MetricsSink) *JSBridge {
	b := &JSBridge{
		Global:    global,
		Local:     nil,
		Sink:      sink,
		IsRuntime: false,
	}
	b.Metrics = &JSMetricsAPI{bridge: b}
	b.LocalAPI = &JSLocalAPI{bridge: b}
	b.GlobalAPI = &JSGlobalAPI{bridge: b}
	return b
}

func (b *JSBridge) AttachWorker(scope runtime.VUContext, local runtime.SharedState, vuId int, iteration int, scenario string) {
	if scope == nil {
		b.IsRuntime = false
	} else {
		b.IsRuntime = true
	}

	b.WorkerScope = scope
	b.Local = local
	b.VuId = vuId
	b.Iteration = iteration
	b.Scenario = scenario
	b.CurrentHook = "unknown"
	b.AbortFlag = false
	b.SleepTime = 0
	b.BarrierName, b.BarrierOptions = "", nil
}

func (b *JSBridge) Load(filepath string) string {
	if !b.IsRuntime {
		return ""
	}
	return filepath
}

func (b *JSBridge) Get(key string) any {
	if !b.IsRuntime || b.WorkerScope == nil {
		return nil
	}
	v, ok := b.WorkerScope.Get(key)
	if ok {
		return v
	}
	return nil
}

func (b *JSBridge) Set(key string, val any) {
	if !b.IsRuntime || b.WorkerScope == nil {
		return
	}
	b.WorkerScope.Set(key, val)
}

func (b *JSBridge) Log(msg string) {
	if !b.IsRuntime || b.Sink == nil {
		return
	}
	b.Sink.Log(b.VuId, b.CurrentHook, "INFO", msg)
}

func (b *JSBridge) Warn(msg string) {
	if !b.IsRuntime || b.Sink == nil {
		return
	}
	b.Sink.Log(b.VuId, b.CurrentHook, "WARN", msg)
}

func (b *JSBridge) Error(msg string) {
	if !b.IsRuntime || b.Sink == nil {
		return
	}
	b.Sink.Log(b.VuId, b.CurrentHook, "ERROR", msg)
}

func (b *JSBridge) Tag(k, v string) {
	if !b.IsRuntime || b.Sink == nil {
		return
	}
	b.Sink.Tag(b.VuId, k, v)
}

func (b *JSBridge) Fail(reason string) {
	if !b.IsRuntime || b.Sink == nil {
		return
	}
	b.Sink.RecordEvent(b.VuId, b.CurrentHook, "FAIL", reason)
}

func (b *JSBridge) Abort(reason string) {
	if !b.IsRuntime {
		return
	}
	b.AbortFlag = true
	if b.Sink != nil {
		b.Sink.RecordEvent(b.VuId, b.CurrentHook, "ABORT", reason)
	}
	panic("BLASTER_ABORT")
}

func (b *JSBridge) Sleep(s float64) {
	if !b.IsRuntime {
		return
	}
	b.SleepTime = int(s * 1000)
	panic("BLASTER_SLEEP")
}

func (b *JSBridge) Barrier(name string, opts map[string]any) {
	if !b.IsRuntime {
		return
	}
	b.BarrierName = name
	b.BarrierOptions = opts
	panic("BLASTER_BARRIER")
}

func (b *JSBridge) Distribute(key string, items []any, fallback any) {
	if !b.IsRuntime || b.Local == nil {
		return
	}
	b.Local.StoreDistribution(key, items, fallback)
}
