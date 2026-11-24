package handler

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/endale-dev/vigil/internal/domain"
	"github.com/endale-dev/vigil/internal/service"
)

func TestUserDTOOmitsCredentials(t *testing.T) {
	user := domain.User{
		ID: "user-1", Name: "Ada", Email: "ada@example.com",
		PasswordHash: "secret-hash", GoogleSub: "google-secret", StatusSlug: "public-status",
		Notify: domain.NotifySettings{EmailAlerts: true, TelegramChatID: "123456"},
	}
	encoded, err := json.Marshal(userDTO(user, true, true, true))
	if err != nil {
		t.Fatal(err)
	}
	result := string(encoded)
	for _, forbidden := range []string{"secret-hash", "google-secret", "passwordHash", "googleSub", "sessionToken"} {
		if strings.Contains(result, forbidden) {
			t.Errorf("user DTO contains sensitive field %q: %s", forbidden, result)
		}
	}
	for _, expected := range []string{`"hasPassword":true`, `"googleLinked":true`, `"google":true`, `"telegram":true`} {
		if !strings.Contains(result, expected) {
			t.Errorf("user DTO is missing %s: %s", expected, result)
		}
	}
}

func TestMonitorAndCheckDTOsUseExplicitUnits(t *testing.T) {
	checkedAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	monitor := monitorDTO(domain.Monitor{
		ID: "monitor-1", OwnerID: "owner-secret", Name: "API", Type: domain.MonitorHTTP,
		Target: "https://api.example", Interval: 90 * time.Second, Timeout: 1500 * time.Millisecond,
		Status: domain.StatusUp, LastCheckedAt: checkedAt, CreatedAt: checkedAt,
	})
	if monitor.IntervalSeconds != 90 || monitor.TimeoutSeconds != 1.5 {
		t.Fatalf("unexpected monitor durations: interval %v seconds, timeout %v seconds", monitor.IntervalSeconds, monitor.TimeoutSeconds)
	}
	if monitor.LastCheckedAt == nil || !monitor.LastCheckedAt.Equal(checkedAt) {
		t.Fatal("monitor last-check timestamp was not mapped")
	}
	encoded, err := json.Marshal(monitor)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "owner-secret") || strings.Contains(string(encoded), "ownerId") {
		t.Fatalf("monitor DTO exposed its owner: %s", encoded)
	}

	check := checkDTO(domain.Check{ID: "check-1", CheckedAt: checkedAt, Up: true, Latency: 1250 * time.Microsecond})
	if check.LatencyMS != 1 {
		t.Fatalf("got %d latency milliseconds, want 1", check.LatencyMS)
	}
}

func TestIncidentAndUptimeDTOsPreserveOptionalValues(t *testing.T) {
	incident := incidentDTO(domain.Incident{ID: "incident-1", StartedAt: time.Now(), Cause: "timeout"})
	if incident.ResolvedAt != nil {
		t.Fatal("open incident should have a null resolvedAt")
	}
	encoded, err := json.Marshal(uptimeSummaryDTO(service.UptimeSummary{}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"ratio":null`) {
		t.Fatalf("empty uptime ratio should be null: %s", encoded)
	}
}

func TestPublicStatusDTOContainsOnlyPublicFields(t *testing.T) {
	checkedAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	status := publicStatusDTO(service.PublicStatus{
		Status: service.OverallDegraded,
		Monitors: []service.PublicStatusMonitor{{
			Name: "Website", Type: domain.MonitorHTTP, Status: domain.StatusDown,
			LastCheckedAt: checkedAt,
			Uptime:        service.MonitorUptime{Last24Hours: service.UptimeSummary{Up: 2, Total: 3}},
			Checks:        []service.PublicCheck{{Up: false, Latency: 15 * time.Millisecond}},
		}},
	})
	encoded, err := json.Marshal(status)
	if err != nil {
		t.Fatal(err)
	}
	result := string(encoded)
	for _, forbidden := range []string{"monitorId", "ownerId", "target", "email", "cause", "detail"} {
		if strings.Contains(result, `"`+forbidden+`"`) {
			t.Errorf("public status contains private field %q: %s", forbidden, result)
		}
	}
	if !strings.Contains(result, `"latencyMs":15`) {
		t.Fatalf("public check latency was not mapped in milliseconds: %s", result)
	}
}
