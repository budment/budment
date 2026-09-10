package http

import (
	"bytes"
	"sync"

	"github.com/budment/budment/internal/fastconv"
	"github.com/goccy/go-json"
	"github.com/tidwall/gjson"
)

var bufferPool = sync.Pool{
	New: func() any { return bytes.NewBuffer(make([]byte, 0, 4096)) },
}

type TraceTimings struct {
	DNSLookup    int64
	TCPConn      int64
	TLSHandshake int64
	TTFB         int64
}

// Response is the "res" object passed to the .after(res) hook.
type Response struct {
	Status  int
	Headers map[string]string
	Body    []byte
	Error   string
	Timings TraceTimings
	buf     *bytes.Buffer
}

func (r *Response) Json(selector ...string) any {
	if len(r.Body) == 0 {
		return nil
	}

	if len(selector) > 0 && selector[0] != "" {
		res := gjson.GetBytes(r.Body, selector[0])
		if !res.Exists() {
			return nil
		}
		return res.Value()
	}

	var out any
	_ = json.Unmarshal(r.Body, &out)
	return out
}

func (r *Response) Contains(substr string) bool {
	if len(r.Body) == 0 || substr == "" {
		return false
	}
	return bytes.Contains(r.Body, fastconv.StringToBytes(substr))
}

func (r *Response) Release() {
	if r.buf != nil {
		r.buf.Reset()
		bufferPool.Put(r.buf)
		r.buf = nil
	}
	r.Body = nil
}

func (r *Response) Assert(expectCode int, expectContains string, extractRules map[string]string, returnCode int) (bool, map[string]any) {
	if expectCode > 0 && returnCode != expectCode {
		return true, nil
	}

	if expectContains != "" && !r.Contains(expectContains) {
		return true, nil
	}

	var extracted map[string]any
	if len(extractRules) > 0 {
		extracted = make(map[string]any, len(extractRules))
		for path, scopeKey := range extractRules {
			if val := r.Json(path); val != nil {
				extracted[scopeKey] = val
			}
		}
	}

	return false, extracted
}
