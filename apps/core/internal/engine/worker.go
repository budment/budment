package engine

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"github.com/vunas/blaster/internal/metrics"
	"github.com/vunas/blaster/internal/planner"
	"github.com/vunas/blaster/internal/runner/http"
	"github.com/vunas/blaster/internal/runtime"
)

type HookExecutor interface {
	ExecuteHook(hookID string, req *http.Request, res *http.Response) error
	EvaluateBoolean(hookID string) (bool, error)
	EvaluateString(hookID string) (string, error)
	IsAborted() bool
	GetSleepTime() int
	GetBarrierInfo() (name string, quorum int)
	GetRetryInfo() (retry bool, delay int, max int, scope string)
	GetFlags() (skip bool, abort bool)
}

type ExecutorPool interface {
	GetExecutor(scope runtime.Scope, local runtime.SharedState, workerID int, iteration int, scenario string) HookExecutor
	PutExecutor(executor HookExecutor)
}

type Worker struct {
	ID             int
	ScenarioName   string
	Scope          *WorkerScope
	EndpointScope  *EndpointScope
	Graph          *planner.Graph
	Pool           ExecutorPool
	Aggregator     *metrics.Aggregator
	BarrierManager *BarrierManager
	HTTPClient     *http.Client
	LocalState     *LocalState
	Sink           runtime.MetricsSink
	Iterations     int
	SharedIters    *int64
	ActiveTarget   *int32
	currentIter    int
}

func NewWorker(id int, graph *planner.Graph, pool ExecutorPool, agg *metrics.Aggregator, sm *BarrierManager, client *http.Client, iterations int, scenarioName string, ls *LocalState, sink runtime.MetricsSink, sharedIters *int64, activeTarget *int32) *Worker {
	return &Worker{
		ID:             id,
		ScenarioName:   scenarioName,
		Scope:          NewWorkerScope(),
		EndpointScope:  NewEndpointScope(),
		Graph:          graph,
		Pool:           pool,
		Aggregator:     agg,
		BarrierManager: sm,
		HTTPClient:     client,
		LocalState:     ls,
		Sink:           sink,
		Iterations:     iterations,
		SharedIters:    sharedIters,
		ActiveTarget:   activeTarget,
	}
}

func fastToString(val any) string {
	if val == nil {
		return ""
	}
	switch v := val.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprint(v)
	}
}

var requestPool = sync.Pool{
	New: func() any {
		return &http.Request{
			Headers: make(map[string]string, 10),
			Body:    make([]byte, 0, 4096),
		}
	},
}

func acquireRequest(method, url string) *http.Request {
	req := requestPool.Get().(*http.Request)
	req.Method = method
	req.URL = url
	req.Body = req.Body[:0]
	return req
}

func releaseRequest(req *http.Request) {
	clear(req.Headers)
	requestPool.Put(req)
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
			if int32(w.ID) > currentTarget {
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
		w.EndpointScope.Reset()
	}
}

func (w *Worker) executeNodes(ctx context.Context, nodes []planner.ExecutableNode) bool {
	for _, node := range nodes {
		switch n := node.(type) {
		case *planner.HttpNode:
			attempts := 0
			maxAttempts := 1

			for attempts < maxAttempts {
				attempts++
				targetURL := n.URL

				jsReq := acquireRequest(n.Method, targetURL)

				for _, inj := range n.Injects {
					var val any

					if inj.IsFromDistribute {
						val = w.LocalState.DistributeNext(inj.StackID)
						w.EndpointScope.Push(inj.StackID, val, 1)
					}

					val = w.EndpointScope.Consume(inj.StackID)
					if val == nil {
						continue
					}

					strVal := fastToString(val)

					if after, ok := strings.CutPrefix(inj.Target, "header."); ok {
						jsReq.Headers[after] = strVal
					} else if strings.Contains(jsReq.URL, "{"+inj.Target+"}") {
						jsReq.URL = strings.ReplaceAll(jsReq.URL, "{"+inj.Target+"}", strVal)
					} else {
						newBody, err := sjson.SetBytes(jsReq.Body, inj.Target, val)
						if err == nil {
							jsReq.Body = newBody
						}
					}
				}

				retryReq, abort := w.runHook(ctx, n.BeforeHookID, jsReq, nil, &maxAttempts)
				if abort {
					releaseRequest(jsReq)
					return true
				}
				if retryReq {
					releaseRequest(jsReq)
					continue
				}

				start := time.Now()
				resp := w.HTTPClient.Do(ctx, jsReq.Method, jsReq.URL, jsReq.Headers, jsReq.Body)

				func() {
					defer resp.Release()
					latency := time.Since(start).Microseconds()
					isSuccess := resp.Status >= 200 && resp.Status < 400 && resp.Error == ""

					path := n.URL
					if idx := strings.Index(path, "://"); idx != -1 {
						if slashIdx := strings.Index(path[idx+3:], "/"); slashIdx != -1 {
							path = path[idx+3+slashIdx:]
						} else {
							path = "/"
						}
					}
					nodeName := n.Method + "   " + path

					w.Aggregator.Metrics.RecordRequest(n.ID, nodeName, isSuccess, latency,
						resp.Timings.TTFB, resp.Timings.TCPConn, resp.Timings.TLSHandshake,
						int64(len(jsReq.Body)), int64(len(resp.Body)), resp.Status)
					if !isSuccess || resp.Error != "" {
						errMsg := fmt.Sprintf("Status: %d, Err: %s", resp.Status, resp.Error)
						w.Aggregator.PushEvent(metrics.MetricEvent{WorkerID: w.ID, NodeID: n.ID, ErrorMsg: errMsg})

						if w.Sink != nil {
							w.Sink.Log(w.ID, n.ID, "SYS_ERR", errMsg)
						}
					}

					jsRes := http.NewResponse(resp.Status, resp.Headers, resp.Body, resp.Error)

					for _, ext := range n.Extracts {
						if res := gjson.GetBytes(resp.Body, ext.Path); res.Exists() {
							if ext.IsDistribute {
								if arr, ok := res.Value().([]any); ok {
									w.LocalState.StoreDistribution(ext.StackID, arr, nil)
								} else {
									w.LocalState.StoreDistribution(ext.StackID, []any{res.Value()}, nil)
								}
							} else {
								w.EndpointScope.Push(ext.StackID, res.Value(), ext.TotalRef)
							}
						}
					}

					retryReq, abort = w.runHook(ctx, n.AfterHookID, jsReq, jsRes, &maxAttempts)
				}()

				releaseRequest(jsReq)

				if abort {
					return true
				}
				if !retryReq {
					break
				}
			}

		case *planner.BranchNode:
			isTrue := w.evaluateCondition(n.ConditionHookID)
			w.Aggregator.Metrics.RecordBranch(n.ID, isTrue)
			if isTrue {
				if abort := w.executeNodes(ctx, n.TruePath); abort {
					return true
				}
			} else {
				if abort := w.executeNodes(ctx, n.FalsePath); abort {
					return true
				}
			}

		case *planner.LoopNode:
			w.Aggregator.Metrics.RecordLoop(n.ID, n.Count)
			for i := int32(0); i < n.Count; i++ {
				w.Scope.Set("loop_index", i)
				if abort := w.executeNodes(ctx, n.Logic); abort {
					return true
				}
			}

		case *planner.ScriptNode:
			start := time.Now()
			_, abort := w.runHook(ctx, n.HookID, nil, nil, nil)
			latency := time.Since(start).Microseconds()

			w.Aggregator.Metrics.RecordScript(n.ID, latency, abort)

			if abort {
				return true
			}

		case *planner.MatchNode:
			inst := w.Pool.GetExecutor(w.Scope, w.LocalState, w.ID, w.currentIter, w.ScenarioName)
			caseVal, _ := inst.EvaluateString(n.ConditionHookID)
			w.Pool.PutExecutor(inst)

			if inst.IsAborted() {
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

		case *planner.PollNode:
			abort := func() bool {
				if n.Interval <= 0 {
					n.Interval = 1 * time.Second
				}
				pollTimer := time.NewTicker(n.Interval)
				defer pollTimer.Stop()

				attempts := int32(0)
				success := false

				for attempts < n.MaxAttempts && !success {
					select {
					case <-ctx.Done():
						return true
					case <-pollTimer.C:
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
							success = true
						}
					}
				}
				w.Aggregator.Metrics.RecordPoll(n.ID, success)
				return false
			}()

			if abort {
				return true
			}
		}
	}
	return false
}

func (w *Worker) runHook(ctx context.Context, hookID string, req *http.Request, res *http.Response, maxAttempts *int) (bool, bool) {
	if hookID == "" {
		return false, false
	}

	var isAborted, retry, skip bool
	var delay, max int
	var scope, barrierName string
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
		retry, delay, max, scope = inst.GetRetryInfo()
		skip, isAborted = inst.GetFlags()
		if inst.IsAborted() {
			isAborted = true
		}
		return nil
	})

	if isAborted {
		return false, true
	}

	if sleepMs > 0 {
		select {
		case <-time.After(time.Duration(sleepMs) * time.Millisecond):
		case <-ctx.Done():
			return false, true
		}
	}

	if barrierName != "" && w.BarrierManager != nil {
		w.BarrierManager.ArriveAndWait(ctx, barrierName, quorum, 0, 0)
	}

	if retry {
		select {
		case <-time.After(time.Duration(delay) * time.Millisecond):
		case <-ctx.Done():
			return false, true
		}
		if scope == "node" && maxAttempts != nil {
			*maxAttempts = max
			return true, false
		}
		return false, false
	}

	if skip {
		return false, false
	}

	return false, false
}

func (w *Worker) evaluateCondition(hookID string) bool {
	inst := w.Pool.GetExecutor(w.Scope, w.LocalState, w.ID, w.currentIter, w.ScenarioName)
	defer w.Pool.PutExecutor(inst)

	result, _ := inst.EvaluateBoolean(hookID)
	return result
}
