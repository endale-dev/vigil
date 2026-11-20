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

type memoryMonitors struct {
	items     map[domain.ID]domain.Monitor
	countErr  error
	createErr error
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
