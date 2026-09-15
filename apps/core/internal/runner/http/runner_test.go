package http

import (
	"context"
	nethttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/budment/budment/internal/template"
)

func TestRunner_RequestPool_Sanitization(t *testing.T) {
	transport := NewSharedTransport(true)
	r := NewRunner(transport)

	// Acquire request from pool and mutate fields
	req := r.AcquireRequest("POST", template.NewExpression("http://example.com/test"), nil)
	httpReq := req.(*Request)
	httpReq.Headers["Authorization"] = "Bearer dirty_data"
	httpReq.Body = append(httpReq.Body, []byte("dirty_body")...)

	// Return request back to pool
	r.ReleaseRequest(req)

	// Re-acquire request; instance must be cleanly reset
	cleanReq := r.AcquireRequest("GET", template.NewExpression("http://example.com/new"), nil)
	cleanHttp := cleanReq.(*Request)

	if cleanHttp.Method != "GET" || cleanHttp.GetTarget() != "http://example.com/new" {
		t.Errorf("method/URL mismatch: expected GET http://example.com/new, got %s %s", cleanHttp.Method, cleanHttp.GetTarget())
	}
	if cleanHttp.Scope != nil {
		t.Errorf("scope not cleared on acquire")
	}
	if len(cleanHttp.Headers) != 0 {
		t.Errorf("headers not cleared on acquire: %v", cleanHttp.Headers)
	}
	if len(cleanHttp.Body) != 0 {
		t.Errorf("body slice not truncated: %d bytes remaining", len(cleanHttp.Body))
	}
	r.ReleaseRequest(cleanReq)
}

func TestRunner_Execute_ResultMapping(t *testing.T) {
	ts := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		w.WriteHeader(nethttp.StatusCreated) // 201
		_, _ = w.Write([]byte(`{"created":true}`))
	}))
	defer ts.Close()

	r := NewRunner(NewSharedTransport(true))
	req := r.AcquireRequest("POST", template.NewExpression(ts.URL), nil)
	defer r.ReleaseRequest(req)

	resp, result := r.Execute(context.Background(), req)
	if resp != nil {
		defer resp.Release()
	}

	// Verify field mapping to engine's runner.ExecutionResult
	if !result.IsSuccess {
		t.Errorf("expected IsSuccess=true for 201 Created, got false")
	}
	if result.Code != 201 {
		t.Errorf("status code mismatch: expected 201, got %d", result.Code)
	}
	if result.BytesIn != int64(len(`{"created":true}`)) {
		t.Errorf("bytes in mismatch: expected %d, got %d", len(`{"created":true}`), result.BytesIn)
	}
	if result.LatencyUs <= 0 {
		t.Errorf("expected positive LatencyUs, got %d", result.LatencyUs)
	}
}
