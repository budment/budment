package http

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"net"
	nethttp "net/http"
	"net/http/cookiejar"
	"net/http/httptrace"
	"strings"
	"time"
)

func NewSharedTransport(insecureSkipVerify bool) *nethttp.Transport {
	return &nethttp.Transport{
		Proxy: nethttp.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
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
}

type Client struct {
	httpClient *nethttp.Client
}

func NewVUClient(transport *nethttp.Transport) *Client {
	jar, _ := cookiejar.New(nil)
	return &Client{
		httpClient: &nethttp.Client{
			Transport: transport,
			Jar:       jar,
			Timeout:   30 * time.Second,
		},
	}
}

func (c *Client) Do(ctx context.Context, method, url string, headers map[string]string, body []byte) *Response {
	var reqBody io.Reader
	if len(body) > 0 {
		reqBody = bytes.NewReader(body)
	}

	req, err := nethttp.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return &Response{Status: 0, Error: "Failed to build request: " + err.Error()}
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	var timings TraceTimings
	var dnsStart, connStart, tlsStart time.Time
	reqStart := time.Now()

	trace := &httptrace.ClientTrace{
		DNSStart:     func(_ httptrace.DNSStartInfo) { dnsStart = time.Now() },
		DNSDone:      func(_ httptrace.DNSDoneInfo) { timings.DNSLookup = time.Since(dnsStart).Microseconds() },
		ConnectStart: func(_, _ string) { connStart = time.Now() },
		ConnectDone: func(net, addr string, err error) {
			if err == nil {
				timings.TCPConn = time.Since(connStart).Microseconds()
			}
		},
		TLSHandshakeStart: func() { tlsStart = time.Now() },
		TLSHandshakeDone: func(_ tls.ConnectionState, err error) {
			if err == nil {
				timings.TLSHandshake = time.Since(tlsStart).Microseconds()
			}
		},
		GotFirstResponseByte: func() { timings.TTFB = time.Since(reqStart).Microseconds() },
	}

	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return &Response{Status: 0, Error: "Network error: " + err.Error()}
	}
	defer resp.Body.Close()

	buf := bufferPool.Get().(*bytes.Buffer)
	_, err = io.Copy(buf, resp.Body)
	if err != nil {
		buf.Reset()
		bufferPool.Put(buf)
		return &Response{Status: resp.StatusCode, Error: "Failed to read body: " + err.Error()}
	}

	respHeaders := make(map[string]string, len(resp.Header))
	for k, v := range resp.Header {
		respHeaders[k] = strings.Join(v, ", ")
	}

	return &Response{
		Status:  resp.StatusCode,
		Headers: respHeaders,
		Body:    buf.Bytes(),
		Timings: timings,
		buf:     buf, // Retained for Release()
	}
}
