package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type mongoPingerStub struct {
	err      error
	deadline time.Time
}

func (stub *mongoPingerStub) Ping(ctx context.Context) error {
	stub.deadline, _ = ctx.Deadline()
	return stub.err
}

func TestHealthHandlerReturnsOKWithinPingDeadline(t *testing.T) {
	pinger := &mongoPingerStub{}
	handler, err := NewHealthHandler(pinger)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	response := httptest.NewRecorder()
	started := time.Now()
	handler.Get(response, request)

	remaining := time.Until(pinger.deadline)
	if response.Code != http.StatusOK || response.Body.String() != "ok" || response.Header().Get("Content-Type") != "text/plain; charset=utf-8" {
		t.Fatalf("health response = %d, %q, %q", response.Code, response.Body.String(), response.Header().Get("Content-Type"))
	}
	if pinger.deadline.IsZero() || pinger.deadline.After(started.Add(2*time.Second)) || remaining <= 0 {
		t.Fatalf("Mongo ping deadline = %v, want within two seconds", pinger.deadline)
	}
}

func TestHealthHandlerReturnsUnavailableWhenPingFails(t *testing.T) {
	handler, err := NewHealthHandler(&mongoPingerStub{err: errors.New("database unavailable")})
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.Get(response, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if response.Code != http.StatusServiceUnavailable || response.Body.String() != "unavailable\n" {
		t.Fatalf("health failure response = %d, %q", response.Code, response.Body.String())
	}
}

func TestNewHealthHandlerRequiresPinger(t *testing.T) {
	if handler, err := NewHealthHandler(nil); err == nil || handler != nil {
		t.Fatalf("NewHealthHandler() = (%v, %v), want dependency error", handler, err)
	}
}
