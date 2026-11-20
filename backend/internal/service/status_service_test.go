package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/endale-dev/vigil/internal/domain"
	apperrors "github.com/endale-dev/vigil/internal/errors"
)

func TestStatusServiceReturnsOnlyPublicMonitorData(t *testing.T) {
	now := time.Date(2027, time.January, 14, 15, 16, 17, 0, time.UTC)
	users := &memoryUsers{stored: map[domain.ID]*domain.User{
		"owner-1": {ID: "owner-1", Email: "private@example.test", StatusSlug: "public-page"},
	}}
	monitors := &memoryMonitors{items: map[domain.ID]domain.Monitor{
		"public-id":  {ID: "public-id", OwnerID: "owner-1", Name: "Public API", Type: domain.MonitorHTTP, Target: "https://secret-target.test", Public: true, Status: domain.StatusUp, LastCheckedAt: now},
		"private-id": {ID: "private-id", OwnerID: "owner-1", Name: "Private API", Target: "https://private-target.test", Public: false},
	}}
	checks := &memoryChecks{
		listedChecks:  []domain.Check{{MonitorID: "public-id", Up: true, Latency: 25 * time.Millisecond, Detail: "private probe detail"}},
		uptimeResults: []domain.Uptime{{Up: 1, Total: 1}, {Up: 5, Total: 6}, {Up: 20, Total: 22}},
	}
	service, err := NewStatusService(users, monitors, checks, &fixedServiceClock{now: now})
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Get(context.Background(), "public-page")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if result.Status != OverallOperational || len(result.Monitors) != 1 {
		t.Fatalf("public status = %+v", result)
	}
	monitor := result.Monitors[0]
	if monitor.Name != "Public API" || monitor.Type != domain.MonitorHTTP || monitor.Status != domain.StatusUp || !monitor.LastCheckedAt.Equal(now) || len(monitor.Checks) != 1 || !monitor.Checks[0].Up || monitor.Checks[0].Latency != 25*time.Millisecond {
		t.Fatalf("public monitor = %+v", monitor)
	}
	if checks.lastLimit != 30 {
		t.Fatalf("public check limit = %d, want 30", checks.lastLimit)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	for _, privateValue := range []string{"private@example.test", "secret-target.test", "private-target.test", "private-id", "public-id", "private probe detail"} {
		if strings.Contains(string(encoded), privateValue) {
			t.Errorf("public status exposed %q: %s", privateValue, encoded)
		}
	}
}

func TestOverallStatus(t *testing.T) {
	for _, test := range []struct {
		name     string
		monitors []domain.Monitor
		want     OverallStatus
	}{
		{name: "no monitors", want: OverallUnknown},
		{name: "unknown monitors", monitors: []domain.Monitor{{Status: domain.StatusUnknown}}, want: OverallUnknown},
		{name: "all up", monitors: []domain.Monitor{{Status: domain.StatusUp}, {Status: domain.StatusUp}}, want: OverallOperational},
		{name: "mixed up and unknown", monitors: []domain.Monitor{{Status: domain.StatusUp}, {Status: domain.StatusUnknown}}, want: OverallUnknown},
		{name: "one down", monitors: []domain.Monitor{{Status: domain.StatusUp}, {Status: domain.StatusDown}}, want: OverallDegraded},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := overallStatus(test.monitors); got != test.want {
				t.Fatalf("overallStatus() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestStatusServiceReturnsNotFoundForUnknownSlug(t *testing.T) {
	service, err := NewStatusService(&memoryUsers{stored: map[domain.ID]*domain.User{}}, &memoryMonitors{}, &memoryChecks{}, &fixedServiceClock{now: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Get(context.Background(), "missing"); err != apperrors.ErrNotFound {
		t.Fatalf("Get() error = %v, want ErrNotFound", err)
	}
}
