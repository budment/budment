package goja

import (
	"fmt"
	"strings"
	"sync"

	"github.com/dop251/goja"
	"github.com/vunas/blaster/internal/runner"
	"github.com/vunas/blaster/internal/runtime"
	"github.com/vunas/blaster/internal/template"
)

type VMInstance struct {
	Runtime        *goja.Runtime
	Bridge         *JSBridge
	invokeHookFunc goja.Callable
	evalBoolFunc   goja.Callable
	evalStrFunc    goja.Callable
}

type Pool struct {
	pool     sync.Pool
	registry *HookRegistry
}

func NewPool(registry *HookRegistry, global runtime.SharedState, sink runtime.MetricsSink) *Pool {
	p := &Pool{
		registry: registry,
	}

	p.pool.New = func() any {
		vm := goja.New()
		vm.SetFieldNameMapper(goja.UncapFieldNameMapper())

		bridge := NewJSBridge(global, sink)

		vm.Set("console", map[string]interface{}{
			"log": func(msg string) {
				if sink != nil {
					sink.Log(bridge.VuId, "console", "INFO", msg)
				}
			},
			"warn": func(msg string) {
				if sink != nil {
					sink.Log(bridge.VuId, "console", "WARN", msg)
				}
			},
			"error": func(msg string) {
				if sink != nil {
					sink.Log(bridge.VuId, "console", "ERROR", msg)
				}
			},
		})

		vm.Set("random", map[string]interface{}{
			"uuid": template.FastUUID,
			"string": func(length int) string {
				return template.FastRandomString(length)
			},
		})

		vm.Set("get", bridge.Get)
		vm.Set("set", bridge.Set)
		vm.Set("load", bridge.Load)
		vm.Set("distribute", bridge.Distribute)

		vm.Set("sleep", bridge.Sleep)
		vm.Set("abort", bridge.Abort)
		vm.Set("fail", bridge.Fail)
		vm.Set("barrier", bridge.Barrier)

		vm.Set("log", bridge.Log)
		vm.Set("warn", bridge.Warn)
		vm.Set("error", bridge.Error)
		vm.Set("tag", bridge.Tag)

		vm.Set("local", bridge.GetLocalNamespace())
		vm.Set("global", bridge.GetGlobalNamespace())
		vm.Set("metrics", map[string]interface{}{
			"trend":   bridge.Metrics.Trend,
			"counter": bridge.Metrics.Counter,
			"gauge":   bridge.Metrics.Gauge,
		})

		vm.Set("__GET_VUID", func() int {
			return bridge.VuId
		})
		vm.Set("__GET_ITER", func() int {
			return bridge.Iteration
		})
		vm.Set("__GET_SCEN", func() string {
			return bridge.Scenario
		})
		_, _ = vm.RunString(`globalThis.info = { get vuId() { return __GET_VUID(); }, get iteration() { return __GET_ITER(); }, get scenario() { return __GET_SCEN(); } };`)

		_, err := vm.RunString(registry.GetCode())
		if err != nil {
			fmt.Printf("[POOL WARNING] Failed to load script into VM: %v\n", err)
		}

		_, _ = vm.RunString(`
			globalThis.__invokeHook = function(id, req, res) {
				if (globalThis.HookRegistry && globalThis.HookRegistry.hooks.has(id)) {
					return globalThis.HookRegistry.hooks.get(id)(req, res); 
				}
			};
			globalThis.__evalBool = function(id) {
				if (globalThis.HookRegistry && globalThis.HookRegistry.hooks.has(id)) { 
					return Boolean(globalThis.HookRegistry.hooks.get(id)());
				} 
				return false;
			};
			globalThis.__evalStr = function(id) {
				if (globalThis.HookRegistry && globalThis.HookRegistry.hooks.has(id)) { 
					return String(globalThis.HookRegistry.hooks.get(id)()); 
				}
				return "";
			};
		`)

		invokeFn, _ := goja.AssertFunction(vm.Get("__invokeHook"))
		evalBoolFn, _ := goja.AssertFunction(vm.Get("__evalBool"))
		evalStrFn, _ := goja.AssertFunction(vm.Get("__evalStr"))

		return &VMInstance{
			Runtime:        vm,
			Bridge:         bridge,
			invokeHookFunc: invokeFn,
			evalBoolFunc:   evalBoolFn,
			evalStrFunc:    evalStrFn,
		}
	}
	return p
}

// Manage the VM pool.
func (p *Pool) GetVM(scope runtime.VUContext, local runtime.SharedState, workerID int, iteration int, scenario string) *VMInstance {
	inst := p.pool.Get().(*VMInstance)
	inst.Bridge.AttachWorker(scope, local, workerID, iteration, scenario)
	return inst
}

func (p *Pool) PutVM(inst *VMInstance) {
	inst.Bridge.AttachWorker(nil, nil, 0, 0, "")
	p.pool.Put(inst)
}

func (inst *VMInstance) ExecuteHook(hookID string, req runner.ProtocolRequest, res runner.ProtocolResponse) (err error) {
	if hookID == "" {
		return nil
	}

	inst.Bridge.CurrentHook = hookID

	defer func() {
		if r := recover(); r != nil {
			errStr := fmt.Sprintf("%v", r)
			if strings.HasPrefix(errStr, "BLASTER_") {
				err = nil
			} else {
				err = fmt.Errorf("JS Panic: %v", r)
			}
		}
	}()

	_, err = inst.invokeHookFunc(
		goja.Undefined(),
		inst.Runtime.ToValue(hookID),
		inst.Runtime.ToValue(req),
		inst.Runtime.ToValue(res),
	)

	return err
}

func (inst *VMInstance) EvaluateBoolean(hookID string) (result bool, err error) {
	inst.Bridge.CurrentHook = hookID

	defer func() {
		if r := recover(); r != nil {
			errStr := fmt.Sprintf("%v", r)
			if !strings.HasPrefix(errStr, "BLASTER_") {
				err = fmt.Errorf("JS Panic: %v", r)
			}
		}
	}()

	v, err := inst.evalBoolFunc(
		goja.Undefined(),
		inst.Runtime.ToValue(hookID),
	)

	if err != nil {
		return false, err
	}
	return v.ToBoolean(), nil
}

func (inst *VMInstance) EvaluateString(hookID string) (result string, err error) {
	inst.Bridge.CurrentHook = hookID

	defer func() {
		if r := recover(); r != nil {
			errStr := fmt.Sprintf("%v", r)
			if !strings.HasPrefix(errStr, "BLASTER_") {
				err = fmt.Errorf("JS Panic: %v", r)
			}
		}
	}()

	v, err := inst.evalStrFunc(
		goja.Undefined(),
		inst.Runtime.ToValue(hookID),
	)

	if err != nil {
		return "", err
	}
	return v.String(), nil
}

func (inst *VMInstance) IsAborted() bool {
	return inst.Bridge.AbortFlag
}

func (inst *VMInstance) GetSleepTime() int {
	return inst.Bridge.SleepTime
}

func (inst *VMInstance) GetBarrierInfo() (string, int) {
	quorum := 10
	if inst.Bridge.BarrierOptions != nil {
		if q, ok := inst.Bridge.BarrierOptions["quorum"].(float64); ok {
			quorum = int(q)
		}
	}
	return inst.Bridge.BarrierName, quorum
}
