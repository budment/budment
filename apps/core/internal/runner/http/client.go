package http

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"sync"
	"time"

	"github.com/vunas/blaster/internal/config"
)

var bufferPool = sync.Pool{
	New: func() any {
		return bytes.NewBuffer(make([]byte, 0, 4096))
	},
}

type TraceTimings struct {
	DNSLookup    int64
	TCPConn      int64
	TLSHandshake int64
	TTFB         int64
}

type HttpResponse struct {
	Status  int
	Headers map[string]string
	Body    []byte
	Error   string
	Timings TraceTimings
	buf     *bytes.Buffer
}

// Release cleans up and returns the buffer to the pool.
func (r *HttpResponse) Release() {
	if r.buf != nil {
		r.buf.Reset()
		bufferPool.Put(r.buf)
		r.buf = nil
		r.Body = nil
	}
}

type Client struct {
	httpClient *http.Client
}

func NewClient(insecureSkipVerify bool, httpCfg config.HTTPConfig) *Client {
	timeoutDur, err := time.ParseDuration(httpCfg.Timeout)
	if err != nil || timeoutDur <= 0 {
		timeoutDur = 30 * time.Second
	}
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   timeoutDur,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100000,
		MaxIdleConnsPerHost:   10000,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: insecureSkipVerify},
	}

	return &Client{
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   timeoutDur,
		},
	}
}

func (c *Client) Do(ctx context.Context, method, url string, headers map[string]string, body []byte) *HttpResponse {
	var reqBody io.Reader
	if len(body) > 0 {
		reqBody = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return &HttpResponse{Status: 0, Error: "Failed to build request: " + err.Error()}
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	var timings TraceTimings
	var dnsStart, connStart, tlsStart, reqStart time.Time

	trace := &httptrace.ClientTrace{
		DNSStart: func(_ httptrace.DNSStartInfo) {
			dnsStart = time.Now()
		},
		DNSDone: func(_ httptrace.DNSDoneInfo) {
			timings.DNSLookup = time.Since(dnsStart).Microseconds()
		},
		ConnectStart: func(_, _ string) {
			connStart = time.Now()
		},
		ConnectDone: func(net, addr string, err error) {
			if err == nil {
				timings.TCPConn = time.Since(connStart).Microseconds()
			}
		},
		TLSHandshakeStart: func() {
			tlsStart = time.Now()
		},
		TLSHandshakeDone: func(_ tls.ConnectionState, err error) {
			if err == nil {
				timings.TLSHandshake = time.Since(tlsStart).Microseconds()
			}
		},
		GotConn: func(_ httptrace.GotConnInfo) {
			reqStart = time.Now()
		},
		GotFirstResponseByte: func() {
			timings.TTFB = time.Since(reqStart).Microseconds()
		},
	}

	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return &HttpResponse{Status: 0, Error: "Network error: " + err.Error()}
	}
	defer resp.Body.Close()

	buf := bufferPool.Get().(*bytes.Buffer)
	_, err = io.Copy(buf, resp.Body)
	if err != nil {
		buf.Reset()
		bufferPool.Put(buf)
		return &HttpResponse{Status: resp.StatusCode, Error: "Failed to read body: " + err.Error()}
	}

	respHeaders := make(map[string]string, len(resp.Header))
	for k, v := range resp.Header {
		if len(v) > 0 {
			respHeaders[k] = v[0]
		}
	}

	return &HttpResponse{
		Status:  resp.StatusCode,
		Headers: respHeaders,
		Body:    buf.Bytes(),
		Timings: timings,
		buf:     buf, // Retained for Release()
	}
}
