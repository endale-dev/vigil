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
	if monitors, err := NewMonitorService(nil, &fixedServiceClock{now: time.Now()}, 1); err == nil || monitors != nil {
		t.Fatalf("NewMonitorService() = (%v, %v), want dependency error", monitors, err)
	}
	if monitors, err := NewMonitorService(&memoryMonitors{}, &fixedServiceClock{now: time.Now()}, 0); err == nil || monitors != nil {
		t.Fatalf("NewMonitorService(limit=0) = (%v, %v), want dependency error", monitors, err)
	}
}

func TestMonitorServiceCreatesMonitorWithDefaults(t *testing.T) {
	clock := &fixedServiceClock{now: time.Date(2026, time.July, 8, 9, 10, 11, 0, time.UTC)}
	repository := &memoryMonitors{}
	monitors, err := NewMonitorService(repository, clock, 25)
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
	monitors, err := NewMonitorService(repository, &fixedServiceClock{now: time.Now()}, 25)
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
	monitors, err := NewMonitorService(repository, &fixedServiceClock{now: time.Now()}, 1)
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
	monitors, err := NewMonitorService(repository, &fixedServiceClock{now: time.Now()}, 25)
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
	monitors, err := NewMonitorService(repository, &fixedServiceClock{now: time.Now()}, 25)
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
	monitors, err := NewMonitorService(repository, &fixedServiceClock{now: now}, 25)
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
	monitors, err := NewMonitorService(repository, &fixedServiceClock{now: time.Now()}, 25)
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
	monitors, err := NewMonitorService(repository, &fixedServiceClock{now: time.Now()}, 25)
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

type memoryMonitors struct {
	items     map[domain.ID]domain.Monitor
	countErr  error
	createErr error
	updateErr error
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

func (*memoryMonitors) RecordResult(context.Context, domain.ID, string, domain.Status, int, time.Time) error {
	return nil
}

var _ domain.MonitorRepository = (*memoryMonitors)(nil)
