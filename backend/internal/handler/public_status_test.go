package handler

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/endale-dev/vigil/internal/domain"
	apperrors "github.com/endale-dev/vigil/internal/errors"
	"github.com/endale-dev/vigil/internal/service"
)

type publicStatusServiceStub struct {
	status service.PublicStatus
	err    error
	slug   string
}

func (stub *publicStatusServiceStub) Get(_ context.Context, slug string) (service.PublicStatus, error) {
	stub.slug = slug
	return stub.status, stub.err
}

func TestPublicStatusHandlerReturnsCachedPublicDTO(t *testing.T) {
	stub := &publicStatusServiceStub{status: service.PublicStatus{
		Status: service.OverallDegraded,
		Monitors: []service.PublicStatusMonitor{{
			Name: "Website", Type: domain.MonitorHTTP, Status: domain.StatusDown,
			LastCheckedAt: time.Date(2026, time.October, 2, 12, 0, 0, 0, time.UTC),
			Uptime:        service.MonitorUptime{Last24Hours: service.UptimeSummary{Up: 8, Total: 10}},
			Checks:        []service.PublicCheck{{Up: false, Latency: 1500 * time.Millisecond}},
		}},
	}}
	handler, err := NewPublicStatusHandler(stub, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/api/public/status/team-status", nil)
	request.SetPathValue("slug", "team-status")
	response := httptest.NewRecorder()
	handler.Get(response, request)

	if response.Code != http.StatusOK || stub.slug != "team-status" || response.Header().Get("Cache-Control") != "public, max-age=15" {
		t.Fatalf("status/slug/cache = %d/%q/%q", response.Code, stub.slug, response.Header().Get("Cache-Control"))
	}
	var result PublicStatusDTO
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Status != string(service.OverallDegraded) || len(result.Monitors) != 1 || result.Monitors[0].Name != "Website" || len(result.Monitors[0].Checks) != 1 || result.Monitors[0].Checks[0].LatencyMS != 1500 {
		t.Fatalf("public status DTO = %+v", result)
	}
	if response.Body.String() == "" || containsAny(response.Body.String(), "ownerId", "monitorId", "detail", "cause", "target") {
		t.Fatalf("public response exposed non-public monitor data: %s", response.Body.String())
	}
}

func TestPublicStatusHandlerDoesNotCacheErrors(t *testing.T) {
	stub := &publicStatusServiceStub{err: apperrors.ErrNotFound}
	handler, err := NewPublicStatusHandler(stub, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/api/public/status/missing", nil)
	request.SetPathValue("slug", "missing")
	response := httptest.NewRecorder()
	handler.Get(response, request)
	if response.Code != http.StatusNotFound || response.Header().Get("Cache-Control") != "" {
		t.Fatalf("error status/cache = %d/%q", response.Code, response.Header().Get("Cache-Control"))
	}
}

func TestNewPublicStatusHandlerRequiresDependencies(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	if handler, err := NewPublicStatusHandler(nil, logger); err == nil || handler != nil {
		t.Fatalf("NewPublicStatusHandler() = (%v, %v), want dependency error", handler, err)
	}
	if handler, err := NewPublicStatusHandler(&publicStatusServiceStub{}, nil); err == nil || handler != nil {
		t.Fatalf("NewPublicStatusHandler() = (%v, %v), want dependency error", handler, err)
	}
}

func containsAny(value string, fragments ...string) bool {
	for _, fragment := range fragments {
		if strings.Contains(value, fragment) {
			return true
		}
	}
	return false
}
