package server

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRecoverHidesAndLogsHandlerPanics(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	handler := RequestLog(logger)(Recover(logger)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("sensitive panic detail")
	})))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/resource", nil))

	if response.Code != http.StatusInternalServerError || response.Body.String() != "{\"error\":\"internal server error\"}\n" {
		t.Fatalf("panic response = %d, %q", response.Code, response.Body.String())
	}
	if strings.Contains(logs.String(), "sensitive panic detail") || !strings.Contains(logs.String(), "HTTP handler panic") || !strings.Contains(logs.String(), "status=500") {
		t.Fatalf("panic log is missing safe recovery details: %s", logs.String())
	}
}

func TestRequestLogIncludesRequestDetailsWithoutQuery(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	handler := RequestLog(logger)(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodPost, "/api/auth/password/reset?token=reset-secret&code=oauth-secret", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	output := logs.String()
	for _, field := range []string{"method=POST", "path=/api/auth/password/reset", "status=204", "duration="} {
		if !strings.Contains(output, field) {
			t.Errorf("request log is missing %q: %s", field, output)
		}
	}
	for _, sensitive := range []string{"reset-secret", "oauth-secret", "?token", "?code"} {
		if strings.Contains(output, sensitive) {
			t.Errorf("request log contains query data %q: %s", sensitive, output)
		}
	}
}

func TestRequestLogDefaultsStatusWhenHandlerWritesBody(t *testing.T) {
	var logs bytes.Buffer
	handler := RequestLog(slog.New(slog.NewTextHandler(&logs, nil)))(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte("ok"))
	}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if response.Code != http.StatusOK || !strings.Contains(logs.String(), "status=200") {
		t.Fatalf("implicit response status/log = %d/%s", response.Code, logs.String())
	}
}

func TestSecurityHeadersSetsConfiguredHeaders(t *testing.T) {
	response := httptest.NewRecorder()
	called := false
	handler := SecurityHeaders(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		called = true
		writer.WriteHeader(http.StatusNoContent)
	}))
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))

	want := map[string]string{
		"Content-Security-Policy": "default-src 'self'; img-src 'self' data:; frame-ancestors 'none'",
		"X-Content-Type-Options":  "nosniff",
		"Referrer-Policy":         "same-origin",
	}
	for name, value := range want {
		if got := response.Header().Get(name); got != value {
			t.Errorf("%s = %q, want %q", name, got, value)
		}
	}
	if !called || response.Code != http.StatusNoContent {
		t.Fatalf("security middleware handler/status = %v/%d", called, response.Code)
	}
}

func TestCSRFAllowsSameOriginBrowserRequests(t *testing.T) {
	for _, test := range []struct {
		name   string
		host   string
		origin string
	}{
		{name: "same origin", host: "vigil.test", origin: "http://vigil.test"},
		{name: "frontend dev proxy", host: "localhost:5173", origin: "http://localhost:5173"},
	} {
		t.Run(test.name, func(t *testing.T) {
			called := false
			handler := CSRF(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				called = true
				writer.WriteHeader(http.StatusNoContent)
			}))
			request := httptest.NewRequest(http.MethodPost, "http://"+test.host+"/api/me/notifications", nil)
			request.Host = test.host
			request.Header.Set("Origin", test.origin)
			request.Header.Set("Sec-Fetch-Site", "same-origin")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if !called || response.Code != http.StatusNoContent {
				t.Fatalf("same-origin request response/handler = %d/%v", response.Code, called)
			}
		})
	}
}

func TestCSRFRejectsCrossOriginCookieStateChanges(t *testing.T) {
	called := false
	handler := CSRF(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	request := httptest.NewRequest(http.MethodPost, "https://vigil.test/api/me/notifications", nil)
	request.Host = "vigil.test"
	request.Header.Set("Origin", "https://attacker.test")
	request.Header.Set("Sec-Fetch-Site", "cross-site")
	request.AddCookie(&http.Cookie{Name: "vigil_session", Value: "session-token"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden || called {
		t.Fatalf("cross-origin state change response/handler = %d/%v, want 403/false", response.Code, called)
	}
}

func TestCSRFAllowsSafeReadsAndNonBrowserRequests(t *testing.T) {
	for _, request := range []*http.Request{
		httptest.NewRequest(http.MethodGet, "https://vigil.test/api/public/status/team", nil),
		httptest.NewRequest(http.MethodPost, "https://vigil.test/api/me/notifications", nil),
	} {
		called := false
		handler := CSRF(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			called = true
			writer.WriteHeader(http.StatusNoContent)
		}))
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if !called || response.Code != http.StatusNoContent {
			t.Fatalf("safe or non-browser request response/handler = %d/%v", response.Code, called)
		}
	}
}
