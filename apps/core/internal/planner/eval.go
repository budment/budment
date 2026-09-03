package planner

import (
	"encoding/json"
	"fmt"
	"sync/atomic"

	"github.com/dop251/goja"
	pb "github.com/vunas/blaster/internal/planner/pb"
	"google.golang.org/protobuf/encoding/protojson"
)

type Evaluator struct{}

func NewEvaluator() *Evaluator {
	return &Evaluator{}
}

func (e *Evaluator) Evaluate(jsBundle string) ([]*pb.Scenario, error) {
	vm := goja.New()
	vm.SetFieldNameMapper(goja.UncapFieldNameMapper())

	e.injectMockDSL(vm)

	// Evaluate the compiled JS bundle.
	_, err := vm.RunString(jsBundle)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate JS bundle: %w", err)
	}

	bridgeScript := `
		if (typeof globalThis.__BLASTER_EXPORTS__ !== 'undefined') {
		  const exp = globalThis.__BLASTER_EXPORTS__;
		  if (exp.default && Array.isArray(exp.default)) {
		    const compilePipeline = (nodes) => {
		      if (!nodes) return undefined;
		      return { steps: nodes.map(n => typeof n.build === 'function' ? n.build() : n) };
		    };
		    const scn = {
		      name: "Default Scenario (K6 Style)",
		      config: exp.options || exp.config || {}, 
		      setup: exp.setup ? compilePipeline(exp.setup) : undefined, 
		      execution: compilePipeline(exp.default) 
		    };
		    globalThis.__BLASTER_SCENARIOS__ = globalThis.__BLASTER_SCENARIOS__ || [];
		    globalThis.__BLASTER_SCENARIOS__.push(scn);
		  }
		}`
	_, _ = vm.RunString(bridgeScript)

	astArrayValue, err := vm.RunString("__BLASTER_SCENARIOS__")
	if err != nil || astArrayValue == nil || goja.IsUndefined(astArrayValue) {
		return nil, fmt.Errorf("no scenarios found. Ensure you use 'export default [...]' or 'scenario().build()'")
	}

	jsonScript := `JSON.stringify(__BLASTER_SCENARIOS__);`
	jsonVal, err := vm.RunString(jsonScript)
	if err != nil {
		return nil, fmt.Errorf("failed to stringify JS AST: %w", err)
	}

	var rawMessages []json.RawMessage
	if err := json.Unmarshal([]byte(jsonVal.String()), &rawMessages); err != nil {
		return nil, fmt.Errorf("failed to parse scenarios array: %w", err)
	}

	// Deserialize the JSON into the Protobuf Scenario.
	var scenarios []*pb.Scenario
	unmarshaler := protojson.UnmarshalOptions{
		DiscardUnknown: true, // Ignore unknown fields for forward compatibility.
	}

	for _, rawMsg := range rawMessages {
		var scn pb.Scenario
		if err := unmarshaler.Unmarshal(rawMsg, &scn); err != nil {
			return nil, fmt.Errorf("failed to unmarshal JSON into Protobuf Scenario: %w", err)
		}
		scenarios = append(scenarios, &scn)
	}

	return scenarios, nil
}

func (e *Evaluator) injectMockDSL(vm *goja.Runtime) {
	var nodeCounter int64 = 0

	genID := func(prefix string) string {
		return fmt.Sprintf("%s_%d", prefix, atomic.AddInt64(&nodeCounter, 1))
	}

	createMockNode := func(nodeType string, data map[string]any) map[string]any {
		return map[string]any{
			"build": func() map[string]any {
				return map[string]any{"id": genID(nodeType), nodeType: data}
			},
		}
	}

	vm.Set("sleep", func(s float64) any { return createMockNode("sleep", map[string]any{"durationS": s}) })
	vm.Set("log", func(msg string) any { return createMockNode("log", map[string]any{"message": msg}) })
	vm.Set("warn", func(msg string) any { return createMockNode("log", map[string]any{"message": "⚠️ [WARN] " + msg}) })
	vm.Set("error", func(msg string) any { return createMockNode("log", map[string]any{"message": "❌ [ERROR] " + msg}) })
	vm.Set("set", func(key string, val any) any {
		return createMockNode("set", map[string]any{"key": key, "valueJson": fmt.Sprint(val), "scope": "worker"})
	})

	vm.Set("barrier", func(name string, opts map[string]any) any {
		quorum := 10
		if opts != nil {
			switch q := opts["quorum"].(type) {
			case int64:
				quorum = int(q)
			case float64:
				quorum = int(q)
			case int:
				quorum = q
			}
		}
		return createMockNode("barrier", map[string]any{"name": name, "quorum": quorum})
	})

	vm.Set("distribute", func(key string, items any, fallback any) any {
		itemsBytes, _ := json.Marshal(items)
		return createMockNode("distribute", map[string]any{"key": key, "itemsJson": string(itemsBytes)})
	})

	metricFunc := func(mType string) func(string, any) any {
		return func(name string, val any) any {
			return createMockNode("metric", map[string]any{"metricType": mType, "name": name, "value": fmt.Sprint(val)})
		}
	}
	vm.Set("metrics", map[string]any{
		"counter": metricFunc("counter"),
		"trend":   metricFunc("trend"),
		"gauge":   metricFunc("gauge"),
	})

	namespaceFunc := func(scope string) map[string]any {
		return map[string]any{
			"get": func(key string) string { return fmt.Sprintf("{{@%s:%s}}", scope, key) },
			"pop": func(queue string) string { return fmt.Sprintf("{{@pop:%s:%s}}", scope, queue) },
			"push": func(queue string, val any) any {
				return createMockNode("log", map[string]any{"message": "Push to " + queue})
			},
			"set": func(key string, val any) any {
				return createMockNode("set", map[string]any{"key": key, "valueJson": fmt.Sprint(val), "scope": scope})
			},
		}
	}
	vm.Set("local", namespaceFunc("local"))
	vm.Set("global", namespaceFunc("global"))

	vm.Set("env", func(key, fallback string) string { return fmt.Sprintf("{{@env:%s:%s}}", key, fallback) })
	vm.Set("open", func(path string) string { return fmt.Sprintf("{{@open:%s}}", path) })
	vm.Set("get", func(key string) string { return fmt.Sprintf("{{%s}}", key) })
	vm.Set("info", map[string]any{"vuId": "{{__VU_ID__}}", "iteration": "{{__ITER__}}", "scenario": "{{__SCENARIO__}}"})

	vm.Set("random", map[string]any{
		"uuid":    func() string { return "{{@random:uuid}}" },
		"string":  func(length int) string { return fmt.Sprintf("{{@random:string:%d}}", length) },
		"integer": func(min, max int) string { return fmt.Sprintf("{{@random:int:%d:%d}}", min, max) },
		"pick": func(arr []any) any {
			if len(arr) > 0 {
				return arr[0]
			}
			return nil
		},
	})
}
