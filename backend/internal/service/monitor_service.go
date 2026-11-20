package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/endale-dev/vigil/internal/domain"
	apperrors "github.com/endale-dev/vigil/internal/errors"
)

type MonitorService struct {
	monitors  domain.MonitorRepository
	checks    domain.CheckRepository
	incidents domain.IncidentRepository
	clock     Clock
	limit     int
}

type UptimeSummary struct {
	Up    int
	Total int
	Ratio *float64
}

type MonitorUptime struct {
	Last24Hours UptimeSummary
	Last7Days   UptimeSummary
	Last30Days  UptimeSummary
}

func NewMonitorService(monitors domain.MonitorRepository, checks domain.CheckRepository, incidents domain.IncidentRepository, clock Clock, limit int) (*MonitorService, error) {
	if monitors == nil || checks == nil || incidents == nil || clock == nil || limit < 1 {
		return nil, errors.New("monitor service dependencies are incomplete")
	}
	return &MonitorService{monitors: monitors, checks: checks, incidents: incidents, clock: clock, limit: limit}, nil
}

func (monitors *MonitorService) Create(ctx context.Context, ownerID domain.ID, draft domain.Monitor) (*domain.Monitor, error) {
	if ownerID == "" {
		return nil, apperrors.ValidationError{Field: "owner", Message: "is required"}
	}
	count, err := monitors.monitors.CountByOwner(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	if count >= monitors.limit {
		return nil, apperrors.ErrLimitReached
	}
	now := monitors.clock.Now()
	monitor := draft
	monitor.ID = ""
	monitor.OwnerID = ownerID
	monitor.Name = strings.TrimSpace(monitor.Name)
	if monitor.Interval == 0 {
		monitor.Interval = time.Minute
	}
	if monitor.Timeout == 0 {
		monitor.Timeout = 10 * time.Second
	}
	if monitor.Type == domain.MonitorTLS && monitor.TLSWarnDays == 0 {
		monitor.TLSWarnDays = 14
	}
	monitor.Status = domain.StatusUnknown
	monitor.Enabled = true
	monitor.ConsecutiveFailures = 0
	monitor.LastCheckedAt = time.Time{}
	monitor.NextCheckAt = now
	monitor.CreatedAt = now
	if err := monitor.Validate(); err != nil {
		return nil, err
	}
	if err := monitors.monitors.Create(ctx, &monitor); err != nil {
		return nil, err
	}
	return &monitor, nil
}

func (monitors *MonitorService) List(ctx context.Context, ownerID domain.ID) ([]domain.Monitor, error) {
	return monitors.monitors.ListByOwner(ctx, ownerID)
}

func (monitors *MonitorService) Get(ctx context.Context, ownerID, monitorID domain.ID) (*domain.Monitor, error) {
	monitor, err := monitors.monitors.GetByID(ctx, monitorID)
	if err != nil {
		return nil, err
	}
	if monitor.OwnerID != ownerID {
		return nil, apperrors.ErrNotFound
	}
	return monitor, nil
}

func (monitors *MonitorService) Update(ctx context.Context, ownerID, monitorID domain.ID, draft domain.Monitor) (*domain.Monitor, error) {
	monitor, err := monitors.Get(ctx, ownerID, monitorID)
	if err != nil {
		return nil, err
	}
	targetChanged := monitor.Type != draft.Type || monitor.Target != draft.Target
	monitor.Name = strings.TrimSpace(draft.Name)
	monitor.Type = draft.Type
	monitor.Target = draft.Target
	monitor.Interval = draft.Interval
	monitor.Timeout = draft.Timeout
	monitor.ExpectedStatus = draft.ExpectedStatus
	monitor.TLSWarnDays = draft.TLSWarnDays
	monitor.Public = draft.Public
	if targetChanged {
		monitor.Status = domain.StatusUnknown
		monitor.ConsecutiveFailures = 0
	}
	if err := monitor.Validate(); err != nil {
		return nil, err
	}
	if err := monitors.monitors.Update(ctx, monitor); err != nil {
		return nil, err
	}
	return monitor, nil
}

func (monitors *MonitorService) Delete(ctx context.Context, ownerID, monitorID domain.ID) error {
	if _, err := monitors.Get(ctx, ownerID, monitorID); err != nil {
		return err
	}
	if err := monitors.checks.DeleteByMonitor(ctx, monitorID); err != nil {
		return err
	}
	if err := monitors.incidents.DeleteByMonitor(ctx, monitorID); err != nil {
		return err
	}
	return monitors.monitors.Delete(ctx, monitorID)
}

func (monitors *MonitorService) Pause(ctx context.Context, ownerID, monitorID domain.ID) error {
	monitor, err := monitors.Get(ctx, ownerID, monitorID)
	if err != nil {
		return err
	}
	monitor.Enabled = false
	return monitors.monitors.Update(ctx, monitor)
}

func (monitors *MonitorService) Resume(ctx context.Context, ownerID, monitorID domain.ID) error {
	monitor, err := monitors.Get(ctx, ownerID, monitorID)
	if err != nil {
		return err
	}
	monitor.Enabled = true
	monitor.NextCheckAt = monitors.clock.Now()
	return monitors.monitors.Update(ctx, monitor)
}

func (monitors *MonitorService) ListChecks(ctx context.Context, ownerID, monitorID domain.ID, limit int) ([]domain.Check, error) {
	if _, err := monitors.Get(ctx, ownerID, monitorID); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 50
	} else if limit > 200 {
		limit = 200
	}
	return monitors.checks.ListByMonitor(ctx, monitorID, limit)
}

func (monitors *MonitorService) ListIncidents(ctx context.Context, ownerID, monitorID domain.ID, limit int) ([]domain.Incident, error) {
	if _, err := monitors.Get(ctx, ownerID, monitorID); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 20
	} else if limit > 50 {
		limit = 50
	}
	return monitors.incidents.ListByMonitor(ctx, monitorID, limit)
}

func (monitors *MonitorService) Uptime(ctx context.Context, ownerID, monitorID domain.ID) (MonitorUptime, error) {
	if _, err := monitors.Get(ctx, ownerID, monitorID); err != nil {
		return MonitorUptime{}, err
	}
	now := monitors.clock.Now()
	periods := []time.Duration{24 * time.Hour, 7 * 24 * time.Hour, 30 * 24 * time.Hour}
	results := make([]UptimeSummary, len(periods))
	for index, period := range periods {
		uptime, err := monitors.checks.UptimeSince(ctx, monitorID, now.Add(-period))
		if err != nil {
			return MonitorUptime{}, err
		}
		results[index] = summarizeUptime(uptime)
	}
	return MonitorUptime{Last24Hours: results[0], Last7Days: results[1], Last30Days: results[2]}, nil
}

func summarizeUptime(uptime domain.Uptime) UptimeSummary {
	summary := UptimeSummary{Up: uptime.Up, Total: uptime.Total}
	if uptime.Total > 0 {
		ratio := float64(uptime.Up) / float64(uptime.Total)
		summary.Ratio = &ratio
	}
	return summary
}
