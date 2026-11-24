package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimitAllowsTenRequestsAndRejectsTheEleventh(t *testing.T) {
	for _, path := range []string{
		"/api/auth/login", "/api/auth/register",
		"/api/auth/password/forgot", "/api/auth/password/reset",
	} {
		t.Run(path, func(t *testing.T) {
			limiter := NewRateLimiter(false)
			handler := RateLimit(limiter)(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.WriteHeader(http.StatusNoContent)
			}))
			for attempt := 1; attempt <= loginRateLimit+1; attempt++ {
				request := httptest.NewRequest(http.MethodPost, path, nil)
				request.RemoteAddr = "192.0.2.10:54321"
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, request)
				if attempt <= loginRateLimit && response.Code != http.StatusNoContent {
					t.Fatalf("request %d returned %d, want 204", attempt, response.Code)
				}
				if attempt == loginRateLimit+1 && response.Code != http.StatusTooManyRequests {
					t.Fatalf("request %d returned %d, want 429", attempt, response.Code)
				}
			}
		})
	}
}

func TestRateLimitSkipsOtherRoutesAndMethods(t *testing.T) {
	limiter := NewRateLimiter(false)
	handler := RateLimit(limiter)(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	}))
	for _, test := range []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/api/auth/login"},
		{method: http.MethodPost, path: "/api/auth/google/callback"},
		{method: http.MethodPost, path: "/api/auth/password/change"},
		{method: http.MethodPost, path: "/api/monitors"},
		{method: http.MethodGet, path: "/api/public/status/team"},
		{method: http.MethodGet, path: "/healthz"},
	} {
		for range loginRateLimit + 1 {
			request := httptest.NewRequest(test.method, test.path, nil)
			request.RemoteAddr = "192.0.2.11:54321"
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusNoContent {
				t.Fatalf("%s %s returned %d, want 204", test.method, test.path, response.Code)
			}
		}
	}
}

func TestRateLimiterBoundsClientsAndResetsExpiredWindows(t *testing.T) {
	limiter := NewRateLimiter(false)
	limiter.maxClients = 1
	now := time.Now()
	if !limiter.allow("192.0.2.1", now) {
		t.Fatal("first client was denied")
	}
	if limiter.allow("192.0.2.2", now) {
		t.Fatal("new client was allowed after the client map reached its limit")
	}
	if !limiter.allow("192.0.2.2", now.Add(loginRateWindow)) {
		t.Fatal("expired client window was not cleaned before admitting a new client")
	}
	if len(limiter.clients) != 1 {
		t.Fatalf("client map contains %d entries, want at most 1", len(limiter.clients))
	}
}

func TestRateLimiterRunCleanupStopsWithContext(t *testing.T) {
	limiter := NewRateLimiter(false)
	limiter.cleanupInterval = 5 * time.Millisecond
	now := time.Now()
	limiter.mu.Lock()
	limiter.clients["expired"] = rateWindow{started: now.Add(-2 * loginRateWindow), count: 1}
	limiter.clients["active"] = rateWindow{started: now, count: 1}
	limiter.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		limiter.RunCleanup(ctx)
		close(done)
	}()
	deadline := time.After(time.Second)
	for {
		limiter.mu.Lock()
		_, expired := limiter.clients["expired"]
		limiter.mu.Unlock()
		if !expired {
			break
		}
		select {
		case <-deadline:
			cancel()
			t.Fatal("cleanup did not remove an expired window")
		case <-time.After(time.Millisecond):
		}
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("cleanup did not stop after context cancellation")
	}
	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	if _, exists := limiter.clients["active"]; !exists {
		t.Fatal("cleanup removed an active client window")
	}
}

func TestClientIPTrustsOnlyFirstForwardedAddressWhenEnabled(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.RemoteAddr = "192.0.2.20:1234"
	request.Header.Set("X-Forwarded-For", "198.51.100.7, 203.0.113.8")
	if got := clientIP(request, false); got != "192.0.2.20" {
		t.Fatalf("clientIP without proxy trust = %q, want remote IP", got)
	}
	if got := clientIP(request, true); got != "198.51.100.7" {
		t.Fatalf("clientIP with proxy trust = %q, want first forwarded IP", got)
	}
	request.Header.Set("X-Forwarded-For", "invalid, 203.0.113.8")
	if got := clientIP(request, true); got != "192.0.2.20" {
		t.Fatalf("clientIP with invalid first forwarded value = %q, want remote IP", got)
	}
}

func TestRateLimitUsesConfiguredProxyTrust(t *testing.T) {
	limiter := NewRateLimiter(true)
	handler := RateLimit(limiter)(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	}))
	for attempt := 1; attempt <= loginRateLimit+1; attempt++ {
		request := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
		request.RemoteAddr = "192.0.2.21:1234"
		request.Header.Set("X-Forwarded-For", "198.51.100.9, 203.0.113.10")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if attempt == loginRateLimit+1 && response.Code != http.StatusTooManyRequests {
			t.Fatalf("request %d returned %d, want 429", attempt, response.Code)
		}
	}
	limiter.mu.Lock()
	_, forwardedClientExists := limiter.clients["198.51.100.9"]
	limiter.mu.Unlock()
	if !forwardedClientExists {
		t.Fatal("trusted forwarded IP was not used as the rate-limit key")
	}
}
