package planner

import (
	"strings"
	"testing"
)

func TestEvaluator_EvaluateDefaultExport(t *testing.T) {
	evaluator := NewEvaluator()

	// Simulate bundled JS script from user TypeScript source
	jsBundle := `
		globalThis.__BUDMENT_EXPORTS__ = {
			options: {
				vus: 10,
				duration: "10s"
			},
			default: [
				sleep(1.0),
				log("executing request")
			]
		};
	`

	scenarios, err := evaluator.Evaluate(jsBundle)
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}

	if len(scenarios) != 1 {
		t.Fatalf("scenarios count mismatch: expected 1, got %d", len(scenarios))
	}

	scn := scenarios[0]
	if scn.Name != "Default Scenario" {
		t.Errorf("scenario name mismatch: expected 'Default Scenario', got '%s'", scn.Name)
	}

	if scn.Execution == nil || len(scn.Execution.Steps) != 2 {
		t.Fatalf("execution pipeline steps count mismatch: expected 2, got %v", scn.Execution)
	}

	// Verify step 0 is SleepNode
	sleepStep := scn.Execution.Steps[0].GetSleep()
	if sleepStep == nil || sleepStep.DurationS != 1.0 {
		t.Errorf("step 0 SleepNode parameter mismatch: got %v", sleepStep)
	}

	// Verify step 1 is LogNode
	logStep := scn.Execution.Steps[1].GetLog()
	if logStep == nil || logStep.Message != "executing request" {
		t.Errorf("step 1 LogNode message mismatch: got %v", logStep)
	}
}

func TestEvaluator_SyntaxError(t *testing.T) {
	evaluator := NewEvaluator()

	// Malformed JavaScript bundle
	invalidBundle := `const a = ;`

	_, err := evaluator.Evaluate(invalidBundle)
	if err == nil {
		t.Fatal("expected syntax error on malformed JS bundle, got nil")
	}
}

func TestEvaluator_DSL_TemplateExpressions(t *testing.T) {
	evaluator := NewEvaluator()

	// Verify template DSL helpers generate valid token syntax {{@...}}
	jsBundle := `
		globalThis.__BUDMENT_EXPORTS__ = {
			default: [
				log(random.uuid()),
				log(env("API_KEY", "default_secret")),
				log(get("user_token"))
			]
		};
	`

	scenarios, err := evaluator.Evaluate(jsBundle)
	if err != nil {
		t.Fatalf("template helpers evaluation failed: %v", err)
	}

	steps := scenarios[0].Execution.Steps
	if steps[0].GetLog().Message != "{{@random:uuid}}" {
		t.Errorf("random.uuid() expression mismatch: expected '{{@random:uuid}}', got '%s'", steps[0].GetLog().Message)
	}
	if steps[1].GetLog().Message != "{{@env:API_KEY:default_secret}}" {
		t.Errorf("env() expression mismatch: expected '{{@env:API_KEY:default_secret}}', got '%s'", steps[1].GetLog().Message)
	}
	if steps[2].GetLog().Message != "{{user_token}}" {
		t.Errorf("get() expression mismatch: expected '{{user_token}}', got '%s'", steps[2].GetLog().Message)
	}
}

func TestEvaluator_DSL_DotNotation_And_ObjectSerialization(t *testing.T) {
	evaluator := NewEvaluator()

	// Simulate DSL script heavily relying on static dot-notation and object state injection
	jsBundle := `
		globalThis.__BUDMENT_EXPORTS__ = {
			default: [
				log(get("user").profile.id),
				log(local.get("config").api.url),
				set("obj", { name: "Alice", age: 25 })
			]
		};
	`

	scenarios, err := evaluator.Evaluate(jsBundle)
	if err != nil {
		t.Fatalf("bundle evaluation failed: %v", err)
	}
	steps := scenarios[0].Execution.Steps

	// 1. Verify Worker Scope proxy translates property access chains to '}{' AST format
	log1 := steps[0].GetLog().Message
	if log1 != "{{user}{profile}{id}}" {
		t.Errorf("worker proxy AST formatting mismatch: expected '{{user}{profile}{id}}', got '%s'", log1)
	}

	// 2. Verify Shared Scope (local/global) proxy translates identically
	log2 := steps[1].GetLog().Message
	if log2 != "{{@local:config}{api}{url}}" {
		t.Errorf("local proxy AST formatting mismatch: expected '{{@local:config}{api}{url}}', got '%s'", log2)
	}

	// 3. Verify object mutations via set() correctly serialize to JSON strings rather than native stringifiers
	setVal := steps[2].GetSet().ValueJson
	if !strings.Contains(setVal, `"name":"Alice"`) {
		t.Errorf("object serialization failed: expected JSON format, got '%s'", setVal)
	}
}
