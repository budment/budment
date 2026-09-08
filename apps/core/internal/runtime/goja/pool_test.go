package goja

import (
	"sync"
	"testing"
)

// newTestRegistry initializes a mock HookRegistry preloaded with test script.
func newTestRegistry(jsCode string) *HookRegistry {
	fullCode := `
		globalThis.HookRegistry = {
			hooks: new Map()
		};
	` + jsCode
	return NewHookRegistry(fullCode)
}

func TestPool_ConcurrentGetPut_Integrity(t *testing.T) {
	// Verify concurrent worker checkout/checkin ensures state isolation without cross-leak
	code := `
		globalThis.HookRegistry.hooks.set("h_add", function() {
			let cur = get("counter") || 0;
			set("counter", cur + 1);
		});
	`
	reg := newTestRegistry(code)
	pool := NewPool(reg, nil, &stubSink{})

	var wg sync.WaitGroup
	workers := 10
	iterationsPerWorker := 20

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			scope := newStubVUContext()

			for it := 0; it < iterationsPerWorker; it++ {
				vm := pool.GetVM(scope, nil, workerID, it, "ConcurrencyScenario")
				err := vm.ExecuteHook("h_add", nil, nil)
				if err != nil {
					t.Errorf("ExecuteHook failed: %v", err)
				}
				pool.PutVM(vm)
			}

			// Ensure worker's isolated scope observed expected increments
			val, _ := scope.Get("counter")
			if val != int64(iterationsPerWorker) && val != iterationsPerWorker {
				t.Errorf("worker %d counter mismatch: expected %d, got %v", workerID, iterationsPerWorker, val)
			}
		}(i)
	}

	wg.Wait()
}

func TestVMInstance_ExecuteHook_SignalsAndPanics(t *testing.T) {
	code := `
		globalThis.HookRegistry.hooks.set("h_sleep", function() {
			sleep(1.5);
		});
		globalThis.HookRegistry.hooks.set("h_abort", function() {
			abort("critical failure");
		});
		globalThis.HookRegistry.hooks.set("h_crash", function() {
			throw new Error("JS runtime error");
		});
	`
	reg := newTestRegistry(code)
	pool := NewPool(reg, nil, &stubSink{})
	vm := pool.GetVM(newStubVUContext(), nil, 1, 0, "Test")
	defer pool.PutVM(vm)

	// Sleep signal: engine catches BLASTER_SLEEP panic, returns err=nil, captures SleepTime
	err := vm.ExecuteHook("h_sleep", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error on BLASTER_SLEEP: %v", err)
	}
	if vm.GetSleepTime() != 1500 {
		t.Fatalf("sleep time mismatch: expected 1500ms, got %d", vm.GetSleepTime())
	}

	// Abort signal: engine catches BLASTER_ABORT panic, returns err=nil, marks IsAborted=true
	err = vm.ExecuteHook("h_abort", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error on BLASTER_ABORT: %v", err)
	}
	if !vm.IsAborted() {
		t.Fatal("expected IsAborted to be true")
	}

	// Runtime JS error (throw Error): must propagate as error
	err = vm.ExecuteHook("h_crash", nil, nil)
	if err == nil {
		t.Fatal("expected error on thrown JS exception, got nil")
	}
}

func TestVMInstance_EvaluateConditions(t *testing.T) {
	code := `
		globalThis.HookRegistry.hooks.set("cond_true", function() { return 10 > 5; });
		globalThis.HookRegistry.hooks.set("cond_false", function() { return "a" === "b"; });
		globalThis.HookRegistry.hooks.set("match_val", function() { return "case_beta"; });
	`
	reg := newTestRegistry(code)
	pool := NewPool(reg, nil, &stubSink{})
	vm := pool.GetVM(newStubVUContext(), nil, 1, 0, "Test")
	defer pool.PutVM(vm)

	// EvaluateBoolean
	isTrue, err := vm.EvaluateBoolean("cond_true")
	if err != nil || !isTrue {
		t.Fatalf("evaluation mismatch: expected true, got %v (err: %v)", isTrue, err)
	}

	isFalse, err := vm.EvaluateBoolean("cond_false")
	if err != nil || isFalse {
		t.Fatalf("evaluation mismatch: expected false, got %v (err: %v)", isFalse, err)
	}

	// EvaluateString
	strVal, err := vm.EvaluateString("match_val")
	if err != nil || strVal != "case_beta" {
		t.Fatalf("evaluation mismatch: expected 'case_beta', got '%s' (err: %v)", strVal, err)
	}
}
