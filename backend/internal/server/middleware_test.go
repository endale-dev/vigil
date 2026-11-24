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
