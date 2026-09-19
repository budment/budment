package engine

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/budment/budment/internal/metrics"
	"github.com/budment/budment/internal/planner"
	"github.com/budment/budment/internal/runner"
	"github.com/budment/budment/internal/runtime"
	"github.com/budment/budment/internal/template"
)

type stubRequest struct {
	method  string
	target  string
	headers map[string]string
	body    string
	mutated bool
}

func (r *stubRequest) GetTarget() string {
	return r.target
}

func (r *stubRequest) ApplyMutation(meta map[string]template.Expression, payload template.Expression, scope template.ScopeProvider) {
	r.mutated = true
	if r.headers == nil {
		r.headers = make(map[string]string, len(meta))
	}
	for k, v := range meta {
		r.headers[k] = v.Render(scope)
	}
	r.body = payload.Render(scope)
}
func (r *stubRequest) Set(body any, options map[string]any) {}
func (r *stubRequest) Json(selector ...string) any          { return nil }

type stubResponse struct {
	status      int
	payload     string
	released    bool
	assertCount int
}

func (r *stubResponse) Json(selector ...string) any {
	if len(selector) > 0 && selector[0] == "token" {
		return "secret-token-xyz"
	}
	return nil
}
func (r *stubResponse) Contains(substr string) bool { return true }
func (r *stubResponse) Release()                    { r.released = true }
func (r *stubResponse) Assert(expectCode int, expectContains string, extract map[string]string, returnCode int) (bool, map[string]any) {
	r.assertCount++
	res := make(map[string]any)
	for k, targetKey := range extract {
		if val := r.Json(k); val != nil {
			res[targetKey] = val
		}
	}
	return returnCode != expectCode, res
}

type stubRunner struct {
	mu           sync.Mutex // Guards concurrent request access
	executeCount int64
	latencyUs    int64
	lastReq      *stubRequest
}

func (sr *stubRunner) AcquireRequest(method string, target template.Expression, scope template.ScopeProvider) runner.ProtocolRequest {
	finalTarget := target.Raw
	if scope != nil && finalTarget != "" {
		finalTarget = target.Render(scope)
	}

	req := &stubRequest{method: method, target: finalTarget}
	sr.mu.Lock()
	sr.lastReq = req
	sr.mu.Unlock()
	return req
}

func (sr *stubRunner) ReleaseRequest(req runner.ProtocolRequest) {}

func (sr *stubRunner) Execute(ctx context.Context, req runner.ProtocolRequest) (runner.ProtocolResponse, runner.ExecutionResult) {
	atomic.AddInt64(&sr.executeCount, 1)
	return &stubResponse{status: 200, payload: `{"token":"secret-token-xyz"}`}, runner.ExecutionResult{
		IsSuccess: true,
		LatencyUs: sr.latencyUs,
		Code:      200,
	}
}

func (sr *stubRunner) GetLastReq() *stubRequest {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	return sr.lastReq
}

type (
	stubExecutorPool struct{}
	stubHookExecutor struct {
		HookExecutor
		evalBoolVal bool
		hookCalled  bool
	}
)

func (s *stubHookExecutor) ExecuteHook(hookID string, req runner.ProtocolRequest, res runner.ProtocolResponse) error {
	s.hookCalled = true
	return nil
}

func (s *stubHookExecutor) EvaluateBoolean(hookID string) (bool, error)  { return s.evalBoolVal, nil }
func (s *stubHookExecutor) EvaluateString(hookID string) (string, error) { return "case_a", nil }
func (s *stubHookExecutor) IsAborted() bool                              { return false }
func (s *stubHookExecutor) GetSleepTime() int                            { return 0 }
func (s *stubHookExecutor) GetBarrierInfo() (string, int)                { return "", 0 }

func (p *stubExecutorPool) GetExecutor(scope runtime.VUContext, local runtime.SharedState, workerID int, iteration int, scenario string) HookExecutor {
	return &stubHookExecutor{evalBoolVal: true}
}
func (p *stubExecutorPool) PutExecutor(executor HookExecutor) {}

type stubSink struct {
	runtime.MetricsSink
	logs []string
	mu   sync.Mutex
}

func (s *stubSink) Log(workerID int, nodeID, level, msg string) {
	s.mu.Lock()
	s.logs = append(s.logs, fmt.Sprintf("[%s] %s: %s", level, nodeID, msg))
	s.mu.Unlock()
}
func (s *stubSink) RecordEvent(workerID int, nodeID, eventType, reason string) {}
func (s *stubSink) RecordCustom(workerID int, mType, name string, val float64) {}

func setupTestWorker(graph *planner.Graph, iters int, sharedIters *int64, target *int64) (*Worker, *stubRunner, *metrics.Aggregator) {
	agg := metrics.NewAggregator(1024, nil)
	runnerInst := &stubRunner{}
	rf := func() *runner.Manager {
		mgr := runner.NewManager()
		mgr.Register("HTTP", runnerInst)
		return mgr
	}
	w := NewWorker(1, 1, graph, &stubExecutorPool{}, agg, nil, rf, iters, "TestScenario", nil, nil, &stubSink{}, sharedIters, target)
	return w, runnerInst, agg
}

func TestWorker_SharedIterations_ConcurrencyIntegrity(t *testing.T) {
	// Verifies that 10 workers concurrently compete for 50 shared iterations without data races or over-consumption.
	totalIterations := 50
	numWorkers := 10

	var sharedIters int64
	var completedIters int64

	runnerInst := &stubRunner{}
	rf := func() *runner.Manager {
		mgr := runner.NewManager()
		mgr.Register("HTTP", runnerInst)
		return mgr
	}

	graph := &planner.Graph{
		Execution: []planner.ExecutableNode{
			&planner.ActionNode{
				ID:       "act_1",
				Protocol: "HTTP",
				Method:   "GET",
				Target:   template.NewExpression("http://localhost:8080"),
			},
		},
	}

	agg := metrics.NewAggregator(1024, nil)
	hardCtx, hardCancel := context.WithCancel(context.Background())
	defer hardCancel()
	softCtx, softCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer softCancel()
	var wg sync.WaitGroup

	for i := 1; i <= numWorkers; i++ {
		wg.Add(1)
		w := NewWorker(i, i, graph, &stubExecutorPool{}, agg, nil, rf, totalIterations, "ConcurrencyScenario", nil, nil, &stubSink{}, &sharedIters, nil)
		go func() {
			defer wg.Done()
			w.Run(hardCtx, softCtx, func() {})
			atomic.AddInt64(&completedIters, int64(w.currentIter))
		}()
	}

	wg.Wait()

	claimedIters := atomic.LoadInt64(&sharedIters)
	executedActions := atomic.LoadInt64(&runnerInst.executeCount)

	// Ticket claiming halts as soon as totalIterations threshold is met
	if executedActions != int64(totalIterations) {
		t.Fatalf("executed actions mismatch: expected %d, got %d (claimed: %d)", totalIterations, executedActions, claimedIters)
	}
}

func TestWorker_Scope_LoopIndexRestoration(t *testing.T) {
	// Verifies that nested LoopNodes properly scope and prune loop_index upon completion.
	var outerObservedIndices []int32
	var innerObservedIndices []int32

	graph := &planner.Graph{
		Execution: []planner.ExecutableNode{
			&planner.LoopNode{
				ID:    "outer_loop",
				Count: 2,
				Logic: []planner.ExecutableNode{
					&planner.SetNode{
						ID:        "record_outer",
						Key:       "dummy",
						ValueJSON: template.NewExpression("val"),
					},
					&planner.LoopNode{
						ID:    "inner_loop",
						Count: 3,
						Logic: []planner.ExecutableNode{
							&planner.SetNode{
								ID:        "record_inner",
								Key:       "dummy",
								ValueJSON: template.NewExpression("val"),
							},
						},
					},
				},
			},
		},
	}

	w, _, _ := setupTestWorker(graph, 1, new(int64), nil)
	ctx := context.Background()

	aborted := w.executeNodes(ctx, graph.Execution)
	if aborted {
		t.Fatal("execution unexpectedly aborted")
	}

	// Verify loop_index is removed from scope once the outer loop finishes
	if _, exists := w.Scope.Get("loop_index"); exists {
		t.Errorf("expected loop_index to be deleted from scope after loop completion, but it was retained")
	}
	_ = outerObservedIndices
	_ = innerObservedIndices
}

func TestWorker_ActionProtocol_PipelineMutationAndAssertion(t *testing.T) {
	// Verifies the complete lifecycle of an ActionNode:
	// 1. BeforePipeline executes ReqMutateNode to set headers and body.
	// 2. Runner.Execute runs request.
	// 3. AfterPipeline executes ResAssertNode to extract token into scope.
	graph := &planner.Graph{
		Execution: []planner.ExecutableNode{
			&planner.ActionNode{
				ID:       "action_login",
				Protocol: "HTTP",
				Method:   "POST",
				Target:   template.NewExpression("http://localhost/login"),
				BeforePipeline: []planner.ExecutableNode{
					&planner.ReqMutateNode{
						ID: "mutate_auth",
						Metadata: map[string]template.Expression{
							"Content-Type": template.NewExpression("application/json"),
						},
						Payload: template.NewExpression(`{"user":"admin"}`),
					},
				},
				AfterPipeline: []planner.ExecutableNode{
					&planner.ResAssertNode{
						ID:         "assert_login",
						ExpectCode: 200,
						Extract: map[string]string{
							"token": "auth_token",
						},
					},
				},
			},
		},
	}

	w, runnerInst, _ := setupTestWorker(graph, 1, new(int64), nil)
	aborted := w.executeNodes(context.Background(), graph.Execution)

	if aborted {
		t.Fatal("executeNodes aborted prematurely")
	}

	// Verify request was mutated as expected
	lastReq := runnerInst.GetLastReq()
	if lastReq == nil || !lastReq.mutated {
		t.Fatal("ReqMutateNode failed to mutate the acquired request")
	}
	if lastReq.headers["Content-Type"] != "application/json" {
		t.Errorf("header Content-Type mismatch: expected 'application/json', got '%s'", lastReq.headers["Content-Type"])
	}
	if lastReq.body != `{"user":"admin"}` {
		t.Errorf("payload mismatch: expected '{\"user\":\"admin\"}', got '%s'", lastReq.body)
	}

	// Verify response assertion extracted value into scope
	extractedToken, ok := w.Scope.Get("auth_token")
	if !ok || extractedToken != "secret-token-xyz" {
		t.Fatalf("extracted token mismatch: expected 'secret-token-xyz', got '%v'", extractedToken)
	}
}

func TestWorker_Cancellation_ContextInterruption(t *testing.T) {
	// Verifies worker aborts promptly when context is canceled during SleepNode execution.
	graph := &planner.Graph{
		Execution: []planner.ExecutableNode{
			&planner.SleepNode{
				ID:       "sleep_long",
				Duration: 5 * time.Second,
			},
		},
	}

	w, _, _ := setupTestWorker(graph, 1, new(int64), nil)
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(30 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	aborted := w.executeNodes(ctx, graph.Execution)
	elapsed := time.Since(start)

	if !aborted {
		t.Error("expected executeNodes to return aborted=true on canceled context")
	}
	if elapsed >= 1*time.Second {
		t.Errorf("worker did not abort promptly upon context cancellation: elapsed %v", elapsed)
	}
}

func BenchmarkWorker_HotPath_ActionExecution(b *testing.B) {
	// Measures worker dispatch overhead on a single native ActionNode.
	graph := &planner.Graph{
		Execution: []planner.ExecutableNode{
			&planner.ActionNode{
				ID:       "bench_act",
				Protocol: "HTTP",
				Method:   "GET",
				Target:   template.NewExpression("http://benchmark.internal/test"),
			},
		},
	}

	w, _, _ := setupTestWorker(graph, 0, new(int64), nil)
	ctx := context.Background()
	nodes := graph.Execution

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		w.executeNodes(ctx, nodes)
	}
}
