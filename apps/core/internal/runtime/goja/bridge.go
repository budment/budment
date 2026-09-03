package goja

import (
	"github.com/vunas/blaster/internal/fastconv"
	"github.com/vunas/blaster/internal/runtime"
)

// Exposes metrics APIs
type JSMetricsAPI struct {
	bridge *JSBridge
}

func (m *JSMetricsAPI) Trend(name string, val any) {
	if !m.bridge.IsRuntime {
		return
	}
	if m.bridge.Sink != nil {
		m.bridge.Sink.RecordCustom(m.bridge.VuId, "trend", name, fastconv.ToFloat64(val))
	}
}

func (m *JSMetricsAPI) Counter(name string, val any) {
	if !m.bridge.IsRuntime {
		return
	}
	if m.bridge.Sink != nil {
		m.bridge.Sink.RecordCustom(m.bridge.VuId, "counter", name, fastconv.ToFloat64(val))
	}
}

func (m *JSMetricsAPI) Gauge(name string, val any) {
	if !m.bridge.IsRuntime {
		return
	}
	if m.bridge.Sink != nil {
		m.bridge.Sink.RecordCustom(m.bridge.VuId, "gauge", name, fastconv.ToFloat64(val))
	}
}

type JSBridge struct {
	WorkerScope    runtime.VUContext
	Local          runtime.SharedState
	Global         runtime.SharedState
	Sink           runtime.MetricsSink
	Metrics        *JSMetricsAPI
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
	b := &JSBridge{Global: global, Local: nil, Sink: sink, IsRuntime: false}
	b.Metrics = &JSMetricsAPI{bridge: b}
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

func (b *JSBridge) GetLocalNamespace() map[string]any {
	return map[string]any{
		"get": func(k string) any {
			if !b.IsRuntime || b.Local == nil {
				return nil
			}
			v, _ := b.Local.Get(k)
			return v
		},
		"set": func(k string, v any) {
			if !b.IsRuntime || b.Local == nil {
				return
			}
			b.Local.Set(k, v)
		},
		"push": func(k string, v any) {
			if !b.IsRuntime || b.Local == nil {
				return
			}
			b.Local.Push(k, v)
		},
		"pop": func(k string) any {
			if !b.IsRuntime || b.Local == nil {
				return nil
			}
			return b.Local.Pop(k)
		},
	}
}

func (b *JSBridge) GetGlobalNamespace() map[string]any {
	return map[string]any{
		"get": func(k string) any {
			if !b.IsRuntime || b.Global == nil {
				return nil
			}
			v, _ := b.Global.Get(k)
			return v
		},
		"set": func(k string, v any) {
			if !b.IsRuntime || b.Global == nil {
				return
			}
			b.Global.Set(k, v)
		},
		"push": func(k string, v any) {
			if !b.IsRuntime || b.Global == nil {
				return
			}
			b.Global.Push(k, v)
		},
		"pop": func(k string) any {
			if !b.IsRuntime || b.Global == nil {
				return nil
			}
			return b.Global.Pop(k)
		},
	}
}
