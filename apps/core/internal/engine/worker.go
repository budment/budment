package engine

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/goccy/go-json"
	"github.com/vunas/blaster/internal/fastconv"
	"github.com/vunas/blaster/internal/metrics"
	"github.com/vunas/blaster/internal/planner"
	"github.com/vunas/blaster/internal/runner"
	"github.com/vunas/blaster/internal/runtime"
)

type HookExecutor interface {
	ExecuteHook(hookID string, req runner.ProtocolRequest, res runner.ProtocolResponse) error
	EvaluateBoolean(hookID string) (bool, error)
	EvaluateString(hookID string) (string, error)
	IsAborted() bool
	GetSleepTime() int
	GetBarrierInfo() (name string, quorum int)
}

type ExecutorPool interface {
	GetExecutor(scope runtime.VUContext, local runtime.SharedState, workerID int, iteration int, scenario string) HookExecutor
	PutExecutor(executor HookExecutor)
}

type Worker struct {
	ID             int
	Slot           int
	ScenarioName   string
	Scope          *WorkerScope
	Graph          *planner.Graph
	Pool           ExecutorPool
	Aggregator     *metrics.Aggregator
	BarrierManager *BarrierManager
	RunnerManager  *runner.Manager
	LocalState     *LocalState
	GlobalState    *GlobalState
	Sink           runtime.MetricsSink
	Iterations     int
	SharedIters    *int64
	ActiveTarget   *int32
	currentIter    int
}

func NewWorker(id int, slot int, graph *planner.Graph, pool ExecutorPool, agg *metrics.Aggregator, sm *BarrierManager, runnerFactory func() *runner.Manager, iterations int, scenarioName string, ls *LocalState, gs *GlobalState, sink runtime.MetricsSink, sharedIters *int64, activeTarget *int32) *Worker {
	return &Worker{
		ID:             id,
		Slot:           slot,
		ScenarioName:   scenarioName,
		Scope:          NewWorkerScope(ls, gs),
		Graph:          graph,
		Pool:           pool,
		Aggregator:     agg,
		BarrierManager: sm,
		RunnerManager:  runnerFactory(),
		LocalState:     ls,
		GlobalState:    gs,
		Sink:           sink,
		Iterations:     iterations,
		SharedIters:    sharedIters,
		ActiveTarget:   activeTarget,
	}
}

func (w *Worker) withExecutor(hookID string, fn func(inst HookExecutor) error) error {
	if hookID == "" {
		return nil
	}
	inst := w.Pool.GetExecutor(w.Scope, w.LocalState, w.ID, w.currentIter, w.ScenarioName)
	defer w.Pool.PutExecutor(inst)
	return fn(inst)
}

func (w *Worker) Run(ctx context.Context, done func()) {
	w.Aggregator.Metrics.AddActiveVU(1)
	defer w.Aggregator.Metrics.AddActiveVU(-1)
	defer done()
	w.currentIter = 0

	for {
		if w.ActiveTarget != nil {
			currentTarget := atomic.LoadInt32(w.ActiveTarget)
			if int32(w.Slot) > currentTarget {
				return
			}
		}

		if w.Iterations > 0 {
			ticket := atomic.AddInt64(w.SharedIters, 1)
			if ticket > int64(w.Iterations) {
				return
			}
		}

		select {
		case <-ctx.Done():
			return
		default:
		}

		w.Scope.Set("__VU_ID__", w.ID)
		w.Scope.Set("__VU_SLOT__", w.Slot)
		w.Scope.Set("__ITER__", w.currentIter)
		w.Scope.Set("__SCENARIO__", w.ScenarioName)

		if w.LocalState != nil {
			keys := w.LocalState.GetAllDistributionKeys()
			for _, k := range keys {
				val := w.LocalState.DistributeNext(k)
				w.Scope.Set(k, val)
			}
		}
		iterStart := time.Now()

		w.executeNodes(ctx, w.Graph.Execution)

		if ctx.Err() != nil {
			return
		}

		iterDuration := time.Since(iterStart).Milliseconds()
		w.Aggregator.Metrics.RecordIteration(iterDuration)
		w.currentIter++
		w.Scope.Reset()
	}
}

func (w *Worker) executeNodes(ctx context.Context, nodes []planner.ExecutableNode) bool {
	for _, node := range nodes {
		switch n := node.(type) {
		case *planner.ActionNode:
			if abort := w.executeActionProtocol(ctx, n); abort {
				return true
			}

		case *planner.BranchNode:
			isTrue := w.evaluateCondition(n.ConditionHookID)
			w.Aggregator.Metrics.RecordBranch(n.ID, isTrue)
			target := n.FalsePath
			if isTrue {
				target = n.TruePath
			}
			if abort := w.executeNodes(ctx, target); abort {
				return true
			}

		case *planner.LoopNode:
			w.Aggregator.Metrics.RecordLoop(n.ID, n.Count)
			for i := int32(0); i < n.Count; i++ {
				if ctx.Err() != nil {
					return true
				}

				w.Scope.Set("loop_index", i)
				if abort := w.executeNodes(ctx, n.Logic); abort {
					return true
				}
			}

		case *planner.PollNode:
			if abort := w.executePoll(ctx, n); abort {
				return true
			}

		case *planner.MatchNode:
			var caseVal string
			var isAborted bool

			w.withExecutor(n.ConditionHookID, func(inst HookExecutor) error {
				caseVal, _ = inst.EvaluateString(n.ConditionHookID)
				isAborted = inst.IsAborted()
				return nil
			})

			if isAborted {
				return true
			}

			w.Aggregator.Metrics.RecordMatch(n.ID, caseVal)
			if targetPath, exists := n.Cases[caseVal]; exists {
				if abort := w.executeNodes(ctx, targetPath); abort {
					return true
				}
			} else if len(n.DefaultPath) > 0 {
				if abort := w.executeNodes(ctx, n.DefaultPath); abort {
					return true
				}
			}

		default:
			if abort := w.executeAuxiliaryNode(ctx, node, nil, nil, 0); abort {
				return true
			}
		}
	}
	return false
}

func (w *Worker) executeActionProtocol(ctx context.Context, n *planner.ActionNode) bool {
	runnerIns := w.RunnerManager.Get(n.Protocol)
	if runnerIns == nil {
		w.Sink.Log(w.ID, n.ID, "SYS_ERR", "Unsupported protocol: "+n.Protocol)
		return true
	}

	target := n.Target.Render(w.Scope)
	req := runnerIns.AcquireRequest(n.Method, target)
	defer runnerIns.ReleaseRequest(req)

	for _, bNode := range n.BeforePipeline {
		if abort := w.executeAuxiliaryNode(ctx, bNode, req, nil, 0); abort {
			return true
		}
	}

	resp, execRes := runnerIns.Execute(ctx, req)
	if resp != nil {
		defer resp.Release()
	}

	w.Aggregator.Metrics.RecordRequest(n.ID, n.Method, target, execRes.IsSuccess, execRes.LatencyUs,
		execRes.TTFBUs, execRes.TCPConnUs, execRes.TLSHandUs,
		execRes.BytesOut, execRes.BytesIn, execRes.Code)

	if !execRes.IsSuccess || execRes.ErrorMessage != "" {
		errMsg := fmt.Sprintf("Status: %d, Err: %s", execRes.Code, execRes.ErrorMessage)
		w.Aggregator.PushEvent(metrics.MetricEvent{WorkerID: w.ID, NodeID: n.ID, ErrorMsg: errMsg})
		if w.Sink != nil {
			w.Sink.Log(w.ID, n.ID, "SYS_ERR", errMsg)
		}
	}

	if resp != nil {
		for _, aNode := range n.AfterPipeline {
			if abort := w.executeAuxiliaryNode(ctx, aNode, req, resp, execRes.Code); abort {
				return true
			}
		}
	}

	return w.Scope.IsAbortFlag()
}

func (w *Worker) executeAuxiliaryNode(ctx context.Context, node planner.ExecutableNode, req runner.ProtocolRequest, res runner.ProtocolResponse, returnCode int) bool {
	switch n := node.(type) {
	case *planner.ScriptNode:
		return w.runHook(ctx, n.HookID, req, res)

	case *planner.SleepNode:
		timer := time.NewTimer(n.Duration)
		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
			return true
		}

	case *planner.LogNode:
		msg := n.Message.Render(w.Scope)
		w.Sink.Log(w.ID, n.ID, "INFO", msg)

	case *planner.BarrierNode:
		if w.BarrierManager != nil {
			w.BarrierManager.ArriveAndWait(ctx, n.Name, n.Quorum, 0)
		}

	case *planner.ReqMutateNode:
		if req != nil {
			opts := make(map[string]any, 1)
			if len(n.Metadata) > 0 {
				headers := make(map[string]any, len(n.Metadata))
				for k, expr := range n.Metadata {
					headers[k] = expr.Render(w.Scope)
				}
				opts["headers"] = headers
			}

			body := n.Payload.Render(w.Scope)
			req.Set(body, opts)
		}

	case *planner.ResAssertNode:
		isFailed := false

		if n.ExpectCode > 0 && returnCode != n.ExpectCode {
			isFailed = true
		}

		if res != nil {
			if n.ExpectBodyContains != "" && !res.Contains(n.ExpectBodyContains) {
				isFailed = true
			}

			if !isFailed {
				for path, scopeKey := range n.Extract {
					if val := res.Json(path); val != nil {
						w.Scope.Set(scopeKey, val)
					}
				}
			}
		}

		if isFailed {
			w.Aggregator.PushEvent(metrics.MetricEvent{
				WorkerID:    w.ID,
				NodeID:      n.ID,
				ErrorMsg:    "Assertion Failed",
				IsLogicFail: true,
			})
		}

	case *planner.SetNode:
		valStr := n.ValueJSON.Render(w.Scope)
		var finalVal any = valStr

		trimmed := strings.TrimSpace(valStr)
		if len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[') {
			var parsed any
			if err := json.Unmarshal(fastconv.StringToBytes(trimmed), &parsed); err == nil {
				finalVal = parsed
			}
		}

		if n.Scope == "global" && w.GlobalState != nil {
			w.GlobalState.Set(n.Key, finalVal)
		} else if n.Scope == "local" && w.LocalState != nil {
			w.LocalState.Set(n.Key, finalVal)
		} else {
			w.Scope.Set(n.Key, finalVal)
		}

	case *planner.DistributeNode:
		valStr := n.ItemsJSON.Render(w.Scope)
		var items []any
		if err := json.Unmarshal(fastconv.StringToBytes(valStr), &items); err != nil {
			items = []any{valStr}
		}
		if w.LocalState != nil {
			w.LocalState.StoreDistribution(n.Key, items, nil)
		}

	case *planner.MetricNode:
		valStr := n.Value.Render(w.Scope)
		valFloat, err := strconv.ParseFloat(valStr, 64)
		if err != nil {
			valFloat = 0
		}
		if w.Sink != nil {
			w.Sink.RecordCustom(w.ID, n.MetricType, n.Name, valFloat)
		}
	}
	return false
}

func (w *Worker) runHook(ctx context.Context, hookID string, req runner.ProtocolRequest, res runner.ProtocolResponse) bool {
	if hookID == "" {
		return false
	}

	var isAborted bool
	var barrierName string
	var quorum, sleepMs int

	w.withExecutor(hookID, func(inst HookExecutor) error {
		err := inst.ExecuteHook(hookID, req, res)
		if err != nil {
			errMsg := fmt.Sprintf("%v", err)
			w.Aggregator.PushEvent(metrics.MetricEvent{WorkerID: w.ID, NodeID: hookID, ErrorMsg: errMsg, IsLogicFail: true})
			if w.Sink != nil {
				w.Sink.Log(w.ID, hookID, "SYS_ERR", errMsg)
			}
			isAborted = true
			return nil
		}

		sleepMs = inst.GetSleepTime()
		barrierName, quorum = inst.GetBarrierInfo()
		isAborted = inst.IsAborted()
		return nil
	})

	if isAborted {
		return true
	}

	if sleepMs > 0 {
		timer := time.NewTimer(time.Duration(sleepMs) * time.Millisecond)
		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
			return true
		}
	}

	if barrierName != "" && w.BarrierManager != nil {
		w.BarrierManager.ArriveAndWait(ctx, barrierName, quorum, 0)
	}

	return false
}

func (w *Worker) evaluateCondition(hookID string) bool {
	var result bool
	w.withExecutor(hookID, func(inst HookExecutor) error {
		result, _ = inst.EvaluateBoolean(hookID)
		return nil
	})
	return result
}

func (w *Worker) executePoll(ctx context.Context, n *planner.PollNode) bool {
	if n.Interval <= 0 {
		n.Interval = 1 * time.Second
	}

	attempts := int32(0)
	pollTimer := time.NewTicker(n.Interval)
	defer pollTimer.Stop()

	for attempts < n.MaxAttempts {
		attempts++

		if abort := w.executeNodes(ctx, n.Logic); abort {
			return true
		}

		var conditionResult bool
		var isAborted bool
		w.withExecutor(n.ConditionHookID, func(inst HookExecutor) error {
			conditionResult, _ = inst.EvaluateBoolean(n.ConditionHookID)
			isAborted = inst.IsAborted()
			return nil
		})

		if isAborted {
			return true
		}
		if conditionResult {
			w.Aggregator.Metrics.RecordPoll(n.ID, true)
			return false
		}

		if attempts >= n.MaxAttempts {
			break
		}

		select {
		case <-ctx.Done():
			return true
		case <-pollTimer.C:
		}
	}

	w.Aggregator.Metrics.RecordPoll(n.ID, false)
	return false
}
