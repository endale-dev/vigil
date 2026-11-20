package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/endale-dev/vigil/internal/domain"
	apperrors "github.com/endale-dev/vigil/internal/errors"
)

func TestNewMonitorServiceRejectsIncompleteDependencies(t *testing.T) {
	if monitors, err := NewMonitorService(nil, &memoryChecks{}, &memoryIncidents{}, &fixedServiceClock{now: time.Now()}, 1); err == nil || monitors != nil {
		t.Fatalf("NewMonitorService() = (%v, %v), want dependency error", monitors, err)
	}
	if monitors, err := NewMonitorService(&memoryMonitors{}, &memoryChecks{}, &memoryIncidents{}, &fixedServiceClock{now: time.Now()}, 0); err == nil || monitors != nil {
		t.Fatalf("NewMonitorService(limit=0) = (%v, %v), want dependency error", monitors, err)
	}
}

func TestMonitorServiceCreatesMonitorWithDefaults(t *testing.T) {
	clock := &fixedServiceClock{now: time.Date(2026, time.July, 8, 9, 10, 11, 0, time.UTC)}
	repository := &memoryMonitors{}
	monitors, err := NewMonitorService(repository, &memoryChecks{}, &memoryIncidents{}, clock, 25)
	if err != nil {
		t.Fatal(err)
	}
	draft := domain.Monitor{
		ID: "ignored-id", OwnerID: "other-owner", Name: "  Example API ", Type: domain.MonitorHTTP,
		Target: "https://example.test/health", Public: true,
	}
	created, err := monitors.Create(context.Background(), "owner-1", draft)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.ID != "monitor-1" || created.OwnerID != "owner-1" || created.Name != "Example API" || created.Interval != time.Minute || created.Timeout != 10*time.Second {
		t.Fatalf("created monitor = %+v", created)
	}
	if !created.Enabled || created.Status != domain.StatusUnknown || created.ConsecutiveFailures != 0 || !created.NextCheckAt.Equal(clock.now) || !created.CreatedAt.Equal(clock.now) {
		t.Fatalf("created monitor initial state = %+v", created)
	}
	if draft.ID != "ignored-id" || draft.OwnerID != "other-owner" || draft.Name != "  Example API " {
		t.Fatal("Create() modified the caller's draft")
	}
}

func TestMonitorServiceCreatesTLSMonitorWithWarningDefault(t *testing.T) {
	repository := &memoryMonitors{}
	monitors, err := NewMonitorService(repository, &memoryChecks{}, &memoryIncidents{}, &fixedServiceClock{now: time.Now()}, 25)
	if err != nil {
		t.Fatal(err)
	}
	created, err := monitors.Create(context.Background(), "owner-1", domain.Monitor{
		Name: "Certificate", Type: domain.MonitorTLS, Target: "example.test:443",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.TLSWarnDays != 14 {
		t.Fatalf("TLSWarnDays = %d, want 14", created.TLSWarnDays)
	}
}

func TestMonitorServiceEnforcesOwnerLimitAndValidation(t *testing.T) {
	repository := &memoryMonitors{items: map[domain.ID]domain.Monitor{
		"existing": {ID: "existing", OwnerID: "owner-1"},
	}}
	monitors, err := NewMonitorService(repository, &memoryChecks{}, &memoryIncidents{}, &fixedServiceClock{now: time.Now()}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if created, err := monitors.Create(context.Background(), "owner-1", domain.Monitor{
		Name: "Another", Type: domain.MonitorHTTP, Target: "https://example.test",
	}); err != apperrors.ErrLimitReached || created != nil {
		t.Fatalf("Create() at limit = (%+v, %v), want ErrLimitReached", created, err)
	}
	if _, err := monitors.Create(context.Background(), "owner-2", domain.Monitor{
		Name: "Invalid", Type: domain.MonitorHTTP, Target: "not a URL",
	}); err == nil {
		t.Fatal("Create() accepted an invalid monitor")
	}
	if len(repository.items) != 1 {
		t.Fatalf("stored monitors = %d, want no invalid writes", len(repository.items))
	}
}

func TestMonitorServicePropagatesCountAndCreateFailures(t *testing.T) {
	want := errors.New("repository unavailable")
	repository := &memoryMonitors{countErr: want}
	monitors, err := NewMonitorService(repository, &memoryChecks{}, &memoryIncidents{}, &fixedServiceClock{now: time.Now()}, 25)
	if err != nil {
		t.Fatal(err)
	}
	draft := domain.Monitor{Name: "API", Type: domain.MonitorHTTP, Target: "https://example.test"}
	if _, err := monitors.Create(context.Background(), "owner-1", draft); !errors.Is(err, want) {
		t.Fatalf("Create() count error = %v, want %v", err, want)
	}
	repository.countErr = nil
	repository.createErr = want
	if _, err := monitors.Create(context.Background(), "owner-1", draft); !errors.Is(err, want) {
		t.Fatalf("Create() create error = %v, want %v", err, want)
	}
}

func TestMonitorServiceListAndGetAreOwnerScoped(t *testing.T) {
	repository := &memoryMonitors{items: map[domain.ID]domain.Monitor{
		"monitor-1": {ID: "monitor-1", OwnerID: "owner-1"},
		"monitor-2": {ID: "monitor-2", OwnerID: "owner-2"},
	}}
	monitors, err := NewMonitorService(repository, &memoryChecks{}, &memoryIncidents{}, &fixedServiceClock{now: time.Now()}, 25)
	if err != nil {
		t.Fatal(err)
	}
	list, err := monitors.List(context.Background(), "owner-1")
	if err != nil || len(list) != 1 || list[0].ID != "monitor-1" {
		t.Fatalf("List(owner-1) = (%+v, %v), want only monitor-1", list, err)
	}
	if _, err := monitors.Get(context.Background(), "owner-1", "monitor-2"); err != apperrors.ErrNotFound {
		t.Fatalf("Get(other owner's monitor) error = %v, want ErrNotFound", err)
	}
	if _, err := monitors.Get(context.Background(), "owner-1", "missing"); err != apperrors.ErrNotFound {
		t.Fatalf("Get(missing monitor) error = %v, want ErrNotFound", err)
	}
}

func TestMonitorServiceUpdateResetsStateWhenTargetChanges(t *testing.T) {
	now := time.Date(2026, time.August, 9, 10, 11, 12, 0, time.UTC)
	existing := domain.Monitor{
		ID: "monitor-1", OwnerID: "owner-1", Name: "Old API", Type: domain.MonitorHTTP,
		Target: "https://old.example.test", Interval: time.Minute, Timeout: 10 * time.Second,
		Public: true, Enabled: false, Status: domain.StatusDown, ConsecutiveFailures: 4,
		CreatedAt: now.Add(-time.Hour), LastCheckedAt: now.Add(-time.Minute),
	}
	repository := &memoryMonitors{items: map[domain.ID]domain.Monitor{"monitor-1": existing}}
	monitors, err := NewMonitorService(repository, &memoryChecks{}, &memoryIncidents{}, &fixedServiceClock{now: now}, 25)
	if err != nil {
		t.Fatal(err)
	}
	updated, err := monitors.Update(context.Background(), "owner-1", "monitor-1", domain.Monitor{
		ID: "ignored-id", OwnerID: "other-owner", Name: " New API ", Type: domain.MonitorHTTP,
		Target: "https://new.example.test", Interval: 2 * time.Minute, Timeout: 5 * time.Second,
		ExpectedStatus: 204, Public: false, Enabled: true, Status: domain.StatusUp,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.ID != existing.ID || updated.OwnerID != existing.OwnerID || updated.Name != "New API" || updated.Interval != 2*time.Minute || updated.Timeout != 5*time.Second || updated.ExpectedStatus != 204 || updated.Public {
		t.Fatalf("updated monitor fields = %+v", updated)
	}
	if updated.Status != domain.StatusUnknown || updated.ConsecutiveFailures != 0 || updated.Enabled || !updated.CreatedAt.Equal(existing.CreatedAt) || !updated.LastCheckedAt.Equal(existing.LastCheckedAt) {
		t.Fatalf("update changed protected state incorrectly: %+v", updated)
	}
	if repository.items[existing.ID] != *updated {
		t.Fatal("updated monitor was not persisted")
	}
}

func TestMonitorServiceUpdatePreservesStatusWithoutTargetChange(t *testing.T) {
	existing := domain.Monitor{
		ID: "monitor-1", OwnerID: "owner-1", Name: "API", Type: domain.MonitorHTTP,
		Target: "https://example.test", Interval: time.Minute, Timeout: 10 * time.Second,
		Status: domain.StatusDown, ConsecutiveFailures: 3,
	}
	repository := &memoryMonitors{items: map[domain.ID]domain.Monitor{existing.ID: existing}}
	monitors, err := NewMonitorService(repository, &memoryChecks{}, &memoryIncidents{}, &fixedServiceClock{now: time.Now()}, 25)
	if err != nil {
		t.Fatal(err)
	}
	updated, err := monitors.Update(context.Background(), "owner-1", existing.ID, existing)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Status != existing.Status || updated.ConsecutiveFailures != existing.ConsecutiveFailures {
		t.Fatalf("Update() reset unchanged target state: %+v", updated)
	}
}

func TestMonitorServiceUpdateHidesOtherOwnersAndRejectsInvalidChanges(t *testing.T) {
	existing := domain.Monitor{
		ID: "monitor-1", OwnerID: "owner-1", Name: "API", Type: domain.MonitorHTTP,
		Target: "https://example.test", Interval: time.Minute, Timeout: 10 * time.Second,
	}
	repository := &memoryMonitors{items: map[domain.ID]domain.Monitor{existing.ID: existing}}
	monitors, err := NewMonitorService(repository, &memoryChecks{}, &memoryIncidents{}, &fixedServiceClock{now: time.Now()}, 25)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := monitors.Update(context.Background(), "owner-2", existing.ID, domain.Monitor{}); err != apperrors.ErrNotFound {
		t.Fatalf("Update(other owner) error = %v, want ErrNotFound", err)
	}
	if _, err := monitors.Update(context.Background(), "owner-1", existing.ID, domain.Monitor{Name: "Bad", Type: domain.MonitorHTTP, Target: "invalid"}); err == nil {
		t.Fatal("Update() accepted invalid monitor fields")
	}
	if repository.items[existing.ID] != existing {
		t.Fatal("invalid or unauthorized update changed monitor")
	}
}

func TestMonitorServiceDeleteRemovesOwnedMonitorAndHistory(t *testing.T) {
	monitor := domain.Monitor{ID: "monitor-1", OwnerID: "owner-1"}
	monitorsRepository := &memoryMonitors{items: map[domain.ID]domain.Monitor{monitor.ID: monitor}}
	checks := &memoryChecks{}
	incidents := &memoryIncidents{}
	monitors, err := NewMonitorService(monitorsRepository, checks, incidents, &fixedServiceClock{now: time.Now()}, 25)
	if err != nil {
		t.Fatal(err)
	}
	if err := monitors.Delete(context.Background(), "owner-1", monitor.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if len(checks.deleted) != 1 || checks.deleted[0] != monitor.ID || len(incidents.deleted) != 1 || incidents.deleted[0] != monitor.ID {
		t.Fatalf("deleted checks/incidents = %v/%v", checks.deleted, incidents.deleted)
	}
	if _, ok := monitorsRepository.items[monitor.ID]; ok {
		t.Fatal("monitor was not deleted")
	}
}

func TestMonitorServiceDeleteHidesOtherOwnersAndStopsOnHistoryErrors(t *testing.T) {
	for _, failure := range []string{"checks", "incidents"} {
		t.Run(failure, func(t *testing.T) {
			monitor := domain.Monitor{ID: "monitor-1", OwnerID: "owner-1"}
			monitorsRepository := &memoryMonitors{items: map[domain.ID]domain.Monitor{monitor.ID: monitor}}
			checks := &memoryChecks{}
			incidents := &memoryIncidents{}
			want := errors.New("history delete failed")
			if failure == "checks" {
				checks.err = want
			} else {
				incidents.err = want
			}
			monitors, err := NewMonitorService(monitorsRepository, checks, incidents, &fixedServiceClock{now: time.Now()}, 25)
			if err != nil {
				t.Fatal(err)
			}
			if err := monitors.Delete(context.Background(), "owner-2", monitor.ID); err != apperrors.ErrNotFound {
				t.Fatalf("Delete(other owner) error = %v, want ErrNotFound", err)
			}
			if len(checks.deleted) != 0 || len(incidents.deleted) != 0 || len(monitorsRepository.items) != 1 {
				t.Fatal("unauthorized delete changed monitor history")
			}
			if err := monitors.Delete(context.Background(), "owner-1", monitor.ID); !errors.Is(err, want) {
				t.Fatalf("Delete() error = %v, want %v", err, want)
			}
			if _, ok := monitorsRepository.items[monitor.ID]; !ok {
				t.Fatal("monitor deleted after history deletion failed")
			}
		})
	}
}

type memoryMonitors struct {
	items           map[domain.ID]domain.Monitor
	countErr        error
	createErr       error
	updateErr       error
	resultErr       error
	resultID        domain.ID
	claimToken      string
	resultStatus    domain.Status
	resultFailures  int
	resultCheckedAt time.Time
}

func TestMonitorServicePauseAndResume(t *testing.T) {
	now := time.Date(2026, time.September, 10, 11, 12, 13, 0, time.UTC)
	originalDue := now.Add(time.Hour)
	repository := &memoryMonitors{items: map[domain.ID]domain.Monitor{
		"monitor-1": {ID: "monitor-1", OwnerID: "owner-1", Enabled: true, NextCheckAt: originalDue},
	}}
	monitors, err := NewMonitorService(repository, &memoryChecks{}, &memoryIncidents{}, &fixedServiceClock{now: now}, 25)
	if err != nil {
		t.Fatal(err)
	}
	if err := monitors.Pause(context.Background(), "owner-1", "monitor-1"); err != nil {
		t.Fatalf("Pause() error = %v", err)
	}
	if repository.items["monitor-1"].Enabled || !repository.items["monitor-1"].NextCheckAt.Equal(originalDue) {
		t.Fatalf("paused monitor = %+v", repository.items["monitor-1"])
	}
	if err := monitors.Resume(context.Background(), "owner-1", "monitor-1"); err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	if !repository.items["monitor-1"].Enabled || !repository.items["monitor-1"].NextCheckAt.Equal(now) {
		t.Fatalf("resumed monitor = %+v", repository.items["monitor-1"])
	}
	if err := monitors.Pause(context.Background(), "owner-2", "monitor-1"); err != apperrors.ErrNotFound {
		t.Fatalf("Pause(other owner) error = %v, want ErrNotFound", err)
	}
}

func TestMonitorServiceBoundsCheckAndIncidentHistory(t *testing.T) {
	monitor := domain.Monitor{ID: "monitor-1", OwnerID: "owner-1"}
	monitorsRepository := &memoryMonitors{items: map[domain.ID]domain.Monitor{monitor.ID: monitor}}
	checks := &memoryChecks{}
	incidents := &memoryIncidents{}
	monitors, err := NewMonitorService(monitorsRepository, checks, incidents, &fixedServiceClock{now: time.Now()}, 25)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		input, wantChecks, wantIncidents int
	}{
		{input: 0, wantChecks: 50, wantIncidents: 20},
		{input: -5, wantChecks: 50, wantIncidents: 20},
		{input: 300, wantChecks: 200, wantIncidents: 50},
		{input: 7, wantChecks: 7, wantIncidents: 7},
	} {
		if _, err := monitors.ListChecks(context.Background(), "owner-1", monitor.ID, test.input); err != nil {
			t.Fatalf("ListChecks(limit=%d) error = %v", test.input, err)
		}
		if _, err := monitors.ListIncidents(context.Background(), "owner-1", monitor.ID, test.input); err != nil {
			t.Fatalf("ListIncidents(limit=%d) error = %v", test.input, err)
		}
		if checks.lastLimit != test.wantChecks || incidents.lastLimit != test.wantIncidents {
			t.Fatalf("limits for input %d = %d/%d, want %d/%d", test.input, checks.lastLimit, incidents.lastLimit, test.wantChecks, test.wantIncidents)
		}
	}
	if _, err := monitors.ListChecks(context.Background(), "owner-2", monitor.ID, 10); err != apperrors.ErrNotFound {
		t.Fatalf("ListChecks(other owner) error = %v, want ErrNotFound", err)
	}
	if _, err := monitors.ListIncidents(context.Background(), "owner-2", monitor.ID, 10); err != apperrors.ErrNotFound {
		t.Fatalf("ListIncidents(other owner) error = %v, want ErrNotFound", err)
	}
}

func TestMonitorServiceCalculatesUptimeForThreePeriods(t *testing.T) {
	now := time.Date(2026, time.October, 11, 12, 13, 14, 0, time.UTC)
	repository := &memoryMonitors{items: map[domain.ID]domain.Monitor{
		"monitor-1": {ID: "monitor-1", OwnerID: "owner-1"},
	}}
	checks := &memoryChecks{uptimeResults: []domain.Uptime{{Up: 18, Total: 20}, {Up: 60, Total: 70}, {Up: 250, Total: 300}}}
	monitors, err := NewMonitorService(repository, checks, &memoryIncidents{}, &fixedServiceClock{now: now}, 25)
	if err != nil {
		t.Fatal(err)
	}
	uptime, err := monitors.Uptime(context.Background(), "owner-1", "monitor-1")
	if err != nil {
		t.Fatalf("Uptime() error = %v", err)
	}
	for index, summary := range []UptimeSummary{uptime.Last24Hours, uptime.Last7Days, uptime.Last30Days} {
		want := []float64{0.9, 60.0 / 70.0, 250.0 / 300.0}[index]
		if summary.Ratio == nil || *summary.Ratio != want {
			t.Errorf("uptime period %d ratio = %v, want %v", index, summary.Ratio, want)
		}
	}
	wantSince := []time.Time{now.Add(-24 * time.Hour), now.Add(-7 * 24 * time.Hour), now.Add(-30 * 24 * time.Hour)}
	if len(checks.uptimeSince) != len(wantSince) {
		t.Fatalf("uptime queries = %d, want %d", len(checks.uptimeSince), len(wantSince))
	}
	for index := range wantSince {
		if !checks.uptimeSince[index].Equal(wantSince[index]) {
			t.Errorf("uptime query %d since = %s, want %s", index, checks.uptimeSince[index], wantSince[index])
		}
	}
}

func TestMonitorServiceReturnsNullUptimeRatiosForEmptyHistory(t *testing.T) {
	repository := &memoryMonitors{items: map[domain.ID]domain.Monitor{
		"monitor-1": {ID: "monitor-1", OwnerID: "owner-1"},
	}}
	monitors, err := NewMonitorService(repository, &memoryChecks{}, &memoryIncidents{}, &fixedServiceClock{now: time.Now()}, 25)
	if err != nil {
		t.Fatal(err)
	}
	uptime, err := monitors.Uptime(context.Background(), "owner-1", "monitor-1")
	if err != nil {
		t.Fatalf("Uptime() error = %v", err)
	}
	for _, summary := range []UptimeSummary{uptime.Last24Hours, uptime.Last7Days, uptime.Last30Days} {
		if summary.Total != 0 || summary.Ratio != nil {
			t.Errorf("empty uptime summary = %+v, want null ratio", summary)
		}
	}
}

func TestMonitorServicePropagatesUptimeErrors(t *testing.T) {
	want := errors.New("uptime query failed")
	checks := &memoryChecks{uptimeErr: want}
	repository := &memoryMonitors{items: map[domain.ID]domain.Monitor{
		"monitor-1": {ID: "monitor-1", OwnerID: "owner-1"},
	}}
	monitors, err := NewMonitorService(repository, checks, &memoryIncidents{}, &fixedServiceClock{now: time.Now()}, 25)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := monitors.Uptime(context.Background(), "owner-1", "monitor-1"); !errors.Is(err, want) {
		t.Fatalf("Uptime() error = %v, want %v", err, want)
	}
}

type memoryChecks struct {
	deleted       []domain.ID
	inserted      []domain.Check
	insertErr     error
	lastLimit     int
	err           error
	uptimeResults []domain.Uptime
	uptimeSince   []time.Time
	uptimeErr     error
}

func (repository *memoryChecks) Insert(_ context.Context, check *domain.Check) error {
	if repository.insertErr != nil {
		return repository.insertErr
	}
	repository.inserted = append(repository.inserted, *check)
	return nil
}

func (repository *memoryChecks) ListByMonitor(_ context.Context, _ domain.ID, limit int) ([]domain.Check, error) {
	repository.lastLimit = limit
	return nil, nil
}

func (repository *memoryChecks) UptimeSince(_ context.Context, _ domain.ID, since time.Time) (domain.Uptime, error) {
	repository.uptimeSince = append(repository.uptimeSince, since)
	if repository.uptimeErr != nil {
		return domain.Uptime{}, repository.uptimeErr
	}
	index := len(repository.uptimeSince) - 1
	if index < len(repository.uptimeResults) {
		return repository.uptimeResults[index], nil
	}
	return domain.Uptime{}, nil
}

func (repository *memoryChecks) DeleteByMonitor(_ context.Context, id domain.ID) error {
	if repository.err != nil {
		return repository.err
	}
	repository.deleted = append(repository.deleted, id)
	return nil
}

type memoryIncidents struct {
	deleted   []domain.ID
	lastLimit int
	err       error
}

func (*memoryIncidents) Open(context.Context, *domain.Incident) error { return nil }

func (*memoryIncidents) Resolve(context.Context, domain.ID, time.Time) error { return nil }

func (repository *memoryIncidents) ListByMonitor(_ context.Context, _ domain.ID, limit int) ([]domain.Incident, error) {
	repository.lastLimit = limit
	return nil, nil
}

func (repository *memoryIncidents) DeleteByMonitor(_ context.Context, id domain.ID) error {
	if repository.err != nil {
		return repository.err
	}
	repository.deleted = append(repository.deleted, id)
	return nil
}

func (repository *memoryMonitors) Create(_ context.Context, monitor *domain.Monitor) error {
	if repository.createErr != nil {
		return repository.createErr
	}
	if repository.items == nil {
		repository.items = make(map[domain.ID]domain.Monitor)
	}
	monitor.ID = domain.ID("monitor-1")
	repository.items[monitor.ID] = *monitor
	return nil
}

func (repository *memoryMonitors) GetByID(_ context.Context, id domain.ID) (*domain.Monitor, error) {
	if monitor, ok := repository.items[id]; ok {
		copyOfMonitor := monitor
		return &copyOfMonitor, nil
	}
	return nil, apperrors.ErrNotFound
}

func (repository *memoryMonitors) ListByOwner(_ context.Context, owner domain.ID) ([]domain.Monitor, error) {
	var result []domain.Monitor
	for _, monitor := range repository.items {
		if monitor.OwnerID == owner {
			result = append(result, monitor)
		}
	}
	return result, nil
}

func (repository *memoryMonitors) ListPublicByOwner(_ context.Context, owner domain.ID) ([]domain.Monitor, error) {
	var result []domain.Monitor
	for _, monitor := range repository.items {
		if monitor.OwnerID == owner && monitor.Public {
			result = append(result, monitor)
		}
	}
	return result, nil
}

func (repository *memoryMonitors) CountByOwner(_ context.Context, owner domain.ID) (int, error) {
	if repository.countErr != nil {
		return 0, repository.countErr
	}
	count := 0
	for _, monitor := range repository.items {
		if monitor.OwnerID == owner {
			count++
		}
	}
	return count, nil
}

func (repository *memoryMonitors) Update(_ context.Context, monitor *domain.Monitor) error {
	if repository.updateErr != nil {
		return repository.updateErr
	}
	if _, ok := repository.items[monitor.ID]; !ok {
		return apperrors.ErrNotFound
	}
	repository.items[monitor.ID] = *monitor
	return nil
}

func (repository *memoryMonitors) Delete(_ context.Context, id domain.ID) error {
	if _, ok := repository.items[id]; !ok {
		return apperrors.ErrNotFound
	}
	delete(repository.items, id)
	return nil
}

func (*memoryMonitors) ClaimDue(context.Context, time.Time) (*domain.MonitorClaim, error) {
	return nil, nil
}

func (repository *memoryMonitors) RecordResult(_ context.Context, id domain.ID, claimToken string, status domain.Status, failures int, checkedAt time.Time) error {
	if repository.resultErr != nil {
		return repository.resultErr
	}
	repository.resultID = id
	repository.claimToken = claimToken
	repository.resultStatus = status
	repository.resultFailures = failures
	repository.resultCheckedAt = checkedAt
	return nil
}

var _ domain.MonitorRepository = (*memoryMonitors)(nil)
var _ domain.CheckRepository = (*memoryChecks)(nil)
var _ domain.IncidentRepository = (*memoryIncidents)(nil)
