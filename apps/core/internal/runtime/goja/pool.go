package goja

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"sync"

	"github.com/dop251/goja"
	"github.com/vunas/blaster/internal/runner/http"
)

// VMInstance holds precompiled callables
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

func NewPool(registry *HookRegistry, global SharedState, local SharedState, sink MetricsSink) *Pool {
	p := &Pool{
		registry: registry,
	}

	p.pool.New = func() any {
		vm := goja.New()
		vm.SetFieldNameMapper(goja.UncapFieldNameMapper())

		bridge := NewJSBridge(global, local, sink)
		vm.Set("ctx", bridge)

		vm.Set("expect", func(val any) map[string]any {
			return map[string]any{
				"toBe": func(expected any) {
					if val != expected {
						bridge.Fail(fmt.Sprintf("Expectation failed: Expected %v, got %v", expected, val))
					}
				},
				"toBeGreaterThan": func(expected float64) {
					vFloat, ok := val.(float64)
					if !ok || vFloat <= expected {
						bridge.Fail(fmt.Sprintf("Expectation failed: %v is not greater than %v", val, expected))
					}
				},
			}
		})

		vm.Set("random", map[string]interface{}{
			"uuid": func() string {
				b := make([]byte, 16)
				_, _ = rand.Read(b)
				b[6] = (b[6] & 0x0f) | 0x40
				b[8] = (b[8] & 0x3f) | 0x80

				var buf [36]byte
				hex.Encode(buf[0:8], b[0:4])
				buf[8] = '-'
				hex.Encode(buf[9:13], b[4:6])
				buf[13] = '-'
				hex.Encode(buf[14:18], b[6:8])
				buf[18] = '-'
				hex.Encode(buf[19:23], b[8:10])
				buf[23] = '-'
				hex.Encode(buf[24:36], b[10:])

				return string(buf[:])
			},
			"string": func(length int) string {
				const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
				b := make([]byte, length)
				for i := range b {
					n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
					b[i] = charset[n.Int64()]
				}
				return string(b)
			},
		})

		vm.Set("console", map[string]interface{}{
			"log":   bridge.Log,
			"warn":  bridge.Warn,
			"error": bridge.Error,
		})

		// Load the compiled scenario.
		_, _ = vm.RunString(registry.GetCode())

		// Precompile helper functions.
		_, _ = vm.RunString(`
			globalThis.__invokeHook = function(id, ctx, req, res) {
				if (globalThis.HookRegistry && globalThis.HookRegistry.hooks.has(id)) {
					return globalThis.HookRegistry.hooks.get(id)(ctx, req, res);
				}
			};
			globalThis.__evalBool = function(id, ctx) {
				if (globalThis.HookRegistry && globalThis.HookRegistry.hooks.has(id)) {
					return Boolean(globalThis.HookRegistry.hooks.get(id)(ctx));
				}
				return false;
			};
			globalThis.__evalStr = function(id, ctx) {
				if (globalThis.HookRegistry && globalThis.HookRegistry.hooks.has(id)) {
					return String(globalThis.HookRegistry.hooks.get(id)(ctx));
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
func (p *Pool) GetVM(scope Scope, local SharedState, workerID int, iteration int, scenario string) *VMInstance {
	inst := p.pool.Get().(*VMInstance)
	inst.Bridge.AttachWorker(scope, local, workerID, iteration, scenario)
	return inst
}

func (p *Pool) PutVM(inst *VMInstance) {
	inst.Bridge.AttachWorker(nil, nil, 0, 0, "")
	p.pool.Put(inst)
}

func (inst *VMInstance) ExecuteHook(hookID string, req *http.Request, res *http.Response) (err error) {
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
		inst.Runtime.ToValue(inst.Bridge),
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
		inst.Runtime.ToValue(inst.Bridge),
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
		inst.Runtime.ToValue(inst.Bridge),
	)

	if err != nil {
		return "", err
	}
	return v.String(), nil
}

func (inst *VMInstance) IsAborted() bool {
	return inst.Bridge.AbortFlag
}
