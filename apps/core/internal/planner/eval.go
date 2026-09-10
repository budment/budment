package planner

import (
	"encoding/json"
	"fmt"
	"sync/atomic"

	pb "github.com/budment/budment/internal/planner/pb"
	"github.com/dop251/goja"
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

	// Dynamic bridge: Supports both 'export default [...]', named exports ('export const scn1 = [...]'),
	// and explicit fluent builder calls ('scenario().config().setup̣().execution()').
	bridgeScript := `
		(() => {
		  let exp = undefined;
		  if (typeof __BUDMENT_EXPORTS__ !== 'undefined' && __BUDMENT_EXPORTS__ && Object.keys(__BUDMENT_EXPORTS__).length > 0) {
		    exp = __BUDMENT_EXPORTS__;
		  } else if (typeof globalThis !== 'undefined' && globalThis.__BUDMENT_EXPORTS__) {
		    exp = globalThis.__BUDMENT_EXPORTS__;
		  }
		
		  globalThis.__BUDMENT_SCENARIOS__ = globalThis.__BUDMENT_SCENARIOS__ || [];
		
		  const compilePipeline = (input) => {
		    if (!input) return undefined;		
		    if (typeof input.build === 'function') {
		      input = input.build();
		    }		
		    if (input && Array.isArray(input.steps)) {
		      return input;
		    }
		    const list = Array.isArray(input) ? input : [input];
		    return {
		      steps: list.map(n => (n && typeof n.build === 'function') ? n.build() : n)
		    };
		  };
		
		  const globalConfig = (exp && (exp.options || exp.config)) || {};
		
		  const registerScenario = (name, val) => {
		    if (val && typeof val.build === 'function') {
		      val = val.build();
		    }
		    if (val && val.execution && val.execution.steps) {
		      if (!val.name) val.name = name;
		      val.config = val.config || globalConfig;
		      globalThis.__BUDMENT_SCENARIOS__.push(val);
		      return;
		    }
		    if (val && typeof val === 'object' && val.execution) {
		      globalThis.__BUDMENT_SCENARIOS__.push({
		        name: val.name || name,
		        config: val.config || globalConfig,
		        setup: val.setup ? compilePipeline(val.setup) : (exp && exp.setup ? compilePipeline(exp.setup) : undefined),
		        execution: compilePipeline(val.execution)
		      });
		      return;
		    }
		    if (Array.isArray(val)) {
		      globalThis.__BUDMENT_SCENARIOS__.push({
		        name: name,
		        config: globalConfig,
		        setup: exp && exp.setup ? compilePipeline(exp.setup) : undefined,
		        execution: compilePipeline(val)
		      });
		    }
		  };
		
		  if (exp) {
		    // Case 1: export default
		    if (exp.default) {
		      if (Array.isArray(exp.default)) {
		        registerScenario("Default Scenario", exp.default);
		      } else {
		        registerScenario("Default Scenario", exp.default);
		      }
		    }
		
		    // Case 2: Multi-scenario named exports
		    for (const key of Object.keys(exp)) {
		      if (key === 'options' || key === 'config' || key === 'setup' || key === 'default') continue;
		      registerScenario(key, exp[key]);
		    }
		  }
		})();`
	if _, err := vm.RunString(bridgeScript); err != nil {
		return nil, fmt.Errorf("bridge script failed: %w", err)
	}

	astArrayValue, err := vm.RunString("globalThis.__BUDMENT_SCENARIOS__ || (typeof __BUDMENT_SCENARIOS__ !== 'undefined' ? __BUDMENT_SCENARIOS__ : undefined)")
	if err != nil || astArrayValue == nil || goja.IsUndefined(astArrayValue) {
		return nil, fmt.Errorf("no scenarios found. Ensure you use 'export default [...]' or 'scenario().build()'")
	}

	jsonScript := `JSON.stringify(globalThis.__BUDMENT_SCENARIOS__ || (typeof __BUDMENT_SCENARIOS__ !== 'undefined' ? __BUDMENT_SCENARIOS__ : []));`
	jsonVal, err := vm.RunString(jsonScript)
	if err != nil {
		return nil, fmt.Errorf("failed to stringify JS AST: %w", err)
	}

	var rawMessages []json.RawMessage
	if err := json.Unmarshal([]byte(jsonVal.String()), &rawMessages); err != nil {
		return nil, fmt.Errorf("failed to parse scenarios array: %w", err)
	}

	if len(rawMessages) == 0 {
		return nil, fmt.Errorf("no scenarios found. Ensure you use 'export default [...]' or 'scenario().build()'")
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
	vm.Set("warn", func(msg string) any { return createMockNode("log", map[string]any{"message": "[WARN] " + msg}) })
	vm.Set("error", func(msg string) any { return createMockNode("log", map[string]any{"message": "[ERROR] " + msg}) })
	vm.Set("tag", func(key string, value string) any {
		return createMockNode("tag", map[string]any{"key": key, "value": value})
	})
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

	vm.Set("open", func(path string, mode ...string) string {
		m := "r"
		if len(mode) > 0 && mode[0] == "b" {
			m = "b"
		}
		return fmt.Sprintf("{{@open:%s:%s}}", path, m)
	})

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
