package http

import (
	"context"
	nethttp "net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClient_CookieJar_SessionPersistence(t *testing.T) {
	// Verify VU persists cookies across subsequent requests (session handling)
	ts := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if r.URL.Path == "/login" {
			nethttp.SetCookie(w, &nethttp.Cookie{Name: "session_id", Value: "secret_cookie_123"})
			w.WriteHeader(nethttp.StatusOK)
			return
		}
		if r.URL.Path == "/profile" {
			cookie, err := r.Cookie("session_id")
			if err != nil || cookie.Value != "secret_cookie_123" {
				w.WriteHeader(nethttp.StatusUnauthorized)
				return
			}
			w.WriteHeader(nethttp.StatusOK)
			return
		}
	}))
	defer ts.Close()

	client := NewVUClient(NewSharedTransport(true))
	ctx := context.Background()

	// Issue login request to receive cookie
	resp1 := client.Do(ctx, "POST", ts.URL+"/login", nil, nil)
	defer resp1.Release()
	if resp1.Status != 200 {
		t.Fatalf("login request failed: expected status 200, got %d", resp1.Status)
	}

	// Issue profile request: CookieJar must automatically forward cookie
	resp2 := client.Do(ctx, "GET", ts.URL+"/profile", nil, nil)
	defer resp2.Release()
	if resp2.Status != 200 {
		t.Fatalf("profile request unauthorized: cookie was not persisted across calls")
	}
}

func TestClient_HTTPTrace_MetricsCapture(t *testing.T) {
	// Verify httptrace captures TTFB > 0 when network delay is simulated
	ts := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		time.Sleep(15 * time.Millisecond) // Simulated latency
		w.WriteHeader(nethttp.StatusOK)
		_, _ = w.Write([]byte("pong"))
	}))
	defer ts.Close()

	client := NewVUClient(NewSharedTransport(true))
	resp := client.Do(context.Background(), "GET", ts.URL, nil, nil)
	defer resp.Release()

	if resp.Status != 200 {
		t.Fatalf("request failed: expected status 200, got %d", resp.Status)
	}
	if resp.Timings.TTFB <= 0 {
		t.Errorf("expected positive TTFB captured via httptrace, got %d µs", resp.Timings.TTFB)
	}
}

func TestClient_NetworkError_Handling(t *testing.T) {
	// Verify graceful error handling on unreachable endpoint without engine panics
	client := NewVUClient(NewSharedTransport(true))
	resp := client.Do(context.Background(), "GET", "http://127.0.0.1:54321/dead-port", nil, nil)

	if resp.Status != 0 {
		t.Errorf("status mismatch on connection error: expected 0, got %d", resp.Status)
	}
	if resp.Error == "" {
		t.Error("expected populated error message on network failure, got empty string")
	}
}
