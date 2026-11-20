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
	service, err := NewCheckService(monitors, checks, prober, &fixedServiceClock{now: now}, 2, slog.New(slog.NewTextHandler(io.Discard, nil)))
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
	service, err := NewCheckService(monitors, checks, &fixedProber{result: ProbeResult{Up: true}}, &fixedServiceClock{now: time.Now()}, 2, slog.New(slog.NewTextHandler(io.Discard, nil)))
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
	service, err := NewCheckService(monitors, checks, &fixedProber{result: ProbeResult{Up: false}}, &fixedServiceClock{now: time.Now()}, 2, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Run(context.Background(), domain.Monitor{ID: "monitor-1", Status: domain.StatusUp}, "stale-token"); !errors.Is(err, want) {
		t.Fatalf("Run() error = %v, want %v", err, want)
	}
	if len(checks.inserted) != 1 {
		t.Fatalf("checks inserted = %d, want 1", len(checks.inserted))
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
