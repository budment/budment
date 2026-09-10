package http

import (
	"context"
	nethttp "net/http"
	"sync"
	"time"

	"github.com/budment/budment/internal/runner"
)

var requestPool = sync.Pool{
	New: func() any {
		return &Request{
			Headers: make(map[string]string, 10),
			Body:    make([]byte, 0, 4096),
		}
	},
}

type Runner struct {
	client *Client
}

func NewRunner(transport *nethttp.Transport) *Runner {
	return &Runner{
		client: NewVUClient(transport),
	}
}

func (r *Runner) AcquireRequest(method, target string) runner.ProtocolRequest {
	req := requestPool.Get().(*Request)
	req.Method = method
	req.URL = target
	req.Body = req.Body[:0]
	clear(req.Headers)
	return req
}

func (r *Runner) ReleaseRequest(req runner.ProtocolRequest) {
	if httpReq, ok := req.(*Request); ok {
		requestPool.Put(httpReq)
	}
}

func (r *Runner) Execute(ctx context.Context, req runner.ProtocolRequest) (runner.ProtocolResponse, runner.ExecutionResult) {
	httpReq, ok := req.(*Request)
	if !ok {
		return nil, runner.ExecutionResult{
			IsSuccess:    false,
			ErrorMessage: "invalid request type: expected *http.Request",
		}
	}

	start := time.Now()
	resp := r.client.Do(ctx, httpReq.Method, httpReq.URL, httpReq.Headers, httpReq.Body)
	latency := time.Since(start).Microseconds()

	execRes := runner.ExecutionResult{
		IsSuccess:    resp.Status >= 200 && resp.Status < 400 && resp.Error == "",
		LatencyUs:    latency,
		TTFBUs:       resp.Timings.TTFB,
		TCPConnUs:    resp.Timings.TCPConn,
		TLSHandUs:    resp.Timings.TLSHandshake,
		BytesOut:     int64(len(httpReq.Body)),
		BytesIn:      int64(len(resp.Body)),
		Code:         resp.Status,
		ErrorMessage: resp.Error,
	}

	return resp, execRes
}
