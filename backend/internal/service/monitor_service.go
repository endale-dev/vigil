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
	monitors domain.MonitorRepository
	clock    Clock
	limit    int
}

func NewMonitorService(monitors domain.MonitorRepository, clock Clock, limit int) (*MonitorService, error) {
	if monitors == nil || clock == nil || limit < 1 {
		return nil, errors.New("monitor service dependencies are incomplete")
	}
	return &MonitorService{monitors: monitors, clock: clock, limit: limit}, nil
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
