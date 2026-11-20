package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/endale-dev/vigil/internal/domain"
)

func TestCheckServiceRunPersistsCheckAndClaimedResult(t *testing.T) {
	now := time.Date(2026, time.November, 12, 13, 14, 15, 0, time.UTC)
	monitor := domain.Monitor{ID: "monitor-1", Status: domain.StatusUp, ConsecutiveFailures: 1}
	monitors := &memoryMonitors{}
	checks := &memoryChecks{}
	prober := &fixedProber{result: ProbeResult{Up: true, Latency: 125 * time.Millisecond, Detail: "HTTP 200"}}
	incidents := &memoryIncidents{}
	service, err := NewCheckService(monitors, checks, incidents, prober, &fixedServiceClock{now: now}, 2, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Run(context.Background(), monitor, "claim-token"); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(checks.inserted) != 1 || checks.inserted[0].MonitorID != monitor.ID || !checks.inserted[0].CheckedAt.Equal(now) || !checks.inserted[0].Up || checks.inserted[0].Latency != 125*time.Millisecond || checks.inserted[0].Detail != "HTTP 200" {
		t.Fatalf("inserted checks = %+v", checks.inserted)
	}
	if monitors.resultID != monitor.ID || monitors.claimToken != "claim-token" || monitors.resultStatus != domain.StatusUp || monitors.resultFailures != 0 || !monitors.resultCheckedAt.Equal(now) {
		t.Fatalf("recorded monitor result = %+v", monitors)
	}
	if prober.monitor != monitor {
		t.Fatalf("probed monitor = %+v, want %+v", prober.monitor, monitor)
	}
}

func TestCheckServiceRunDoesNotRecordResultWhenCheckInsertFails(t *testing.T) {
	want := errors.New("check store unavailable")
	monitors := &memoryMonitors{}
	checks := &memoryChecks{insertErr: want}
	service, err := NewCheckService(monitors, checks, &memoryIncidents{}, &fixedProber{result: ProbeResult{Up: true}}, &fixedServiceClock{now: time.Now()}, 2, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Run(context.Background(), domain.Monitor{ID: "monitor-1"}, "claim-token"); !errors.Is(err, want) {
		t.Fatalf("Run() error = %v, want %v", err, want)
	}
	if monitors.resultID != "" {
		t.Fatal("monitor result recorded after check insert failed")
	}
}

func TestCheckServiceRunReturnsResultPersistenceFailure(t *testing.T) {
	want := errors.New("claim expired")
	monitors := &memoryMonitors{resultErr: want}
	checks := &memoryChecks{}
	incidents := &memoryIncidents{}
	service, err := NewCheckService(monitors, checks, incidents, &fixedProber{result: ProbeResult{Up: false}}, &fixedServiceClock{now: time.Now()}, 2, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Run(context.Background(), domain.Monitor{ID: "monitor-1", Status: domain.StatusUp}, "stale-token"); !errors.Is(err, want) {
		t.Fatalf("Run() error = %v, want %v", err, want)
	}
	if len(checks.inserted) != 1 {
		t.Fatalf("checks inserted = %d, want 1", len(checks.inserted))
	}
	if len(incidents.opened) != 0 || len(incidents.resolved) != 0 {
		t.Fatal("incident changed after monitor result persistence failed")
	}
}

func TestCheckServiceOpensAndResolvesIncidentsAfterResultPersistence(t *testing.T) {
	now := time.Date(2026, time.December, 13, 14, 15, 16, 0, time.UTC)
	incidents := &memoryIncidents{}
	clock := &fixedServiceClock{now: now}
	monitors := &memoryMonitors{}
	checks := &memoryChecks{}
	service, err := NewCheckService(monitors, checks, incidents, &fixedProber{result: ProbeResult{Detail: "connection refused"}}, clock, 2, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	down := domain.Monitor{ID: "monitor-1", Status: domain.StatusUp, ConsecutiveFailures: 1}
	if err := service.Run(context.Background(), down, "claim-down"); err != nil {
		t.Fatalf("Run(down) error = %v", err)
	}
	if len(incidents.opened) != 1 || incidents.opened[0].MonitorID != down.ID || incidents.opened[0].Cause != "connection refused" || !incidents.opened[0].StartedAt.Equal(now) {
		t.Fatalf("opened incidents = %+v", incidents.opened)
	}
	monitors.resultStatus = domain.StatusDown
	service.prober = &fixedProber{result: ProbeResult{Up: true, Detail: "HTTP 200"}}
	if err := service.Run(context.Background(), domain.Monitor{ID: down.ID, Status: domain.StatusDown, ConsecutiveFailures: 2}, "claim-up"); err != nil {
		t.Fatalf("Run(recovery) error = %v", err)
	}
	if len(incidents.resolved) != 1 || incidents.resolved[0] != down.ID || !incidents.resolvedAt[0].Equal(now) {
		t.Fatalf("resolved incidents = %v at %v", incidents.resolved, incidents.resolvedAt)
	}
}

func TestCheckServiceDoesNotCreateIncidentBeforeFailureThreshold(t *testing.T) {
	incidents := &memoryIncidents{}
	service, err := NewCheckService(&memoryMonitors{}, &memoryChecks{}, incidents, &fixedProber{result: ProbeResult{Detail: "timeout"}}, &fixedServiceClock{now: time.Now()}, 2, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Run(context.Background(), domain.Monitor{ID: "monitor-1", Status: domain.StatusUp}, "claim-token"); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(incidents.opened) != 0 {
		t.Fatalf("opened incidents before threshold = %+v", incidents.opened)
	}
}

type fixedProber struct {
	result  ProbeResult
	monitor domain.Monitor
}

func (prober *fixedProber) Probe(_ context.Context, monitor domain.Monitor) ProbeResult {
	prober.monitor = monitor
	return prober.result
}
