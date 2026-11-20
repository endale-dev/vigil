package service

import (
	"context"
	"errors"
	"time"

	"github.com/endale-dev/vigil/internal/domain"
)

type OverallStatus string

const (
	OverallOperational OverallStatus = "operational"
	OverallDegraded    OverallStatus = "degraded"
	OverallUnknown     OverallStatus = "unknown"
)

type PublicCheck struct {
	Up      bool
	Latency time.Duration
}

type PublicStatusMonitor struct {
	Name          string
	Type          domain.MonitorType
	Status        domain.Status
	LastCheckedAt time.Time
	Uptime        MonitorUptime
	Checks        []PublicCheck
}

type PublicStatus struct {
	Status   OverallStatus
	Monitors []PublicStatusMonitor
}

type StatusService struct {
	users    domain.UserRepository
	monitors domain.MonitorRepository
	checks   domain.CheckRepository
	clock    Clock
}

func NewStatusService(users domain.UserRepository, monitors domain.MonitorRepository, checks domain.CheckRepository, clock Clock) (*StatusService, error) {
	if users == nil || monitors == nil || checks == nil || clock == nil {
		return nil, errors.New("status service dependencies are incomplete")
	}
	return &StatusService{users: users, monitors: monitors, checks: checks, clock: clock}, nil
}

func (status *StatusService) Get(ctx context.Context, slug string) (PublicStatus, error) {
	user, err := status.users.GetBySlug(ctx, slug)
	if err != nil {
		return PublicStatus{}, err
	}
	monitors, err := status.monitors.ListPublicByOwner(ctx, user.ID)
	if err != nil {
		return PublicStatus{}, err
	}
	response := PublicStatus{
		Status:   overallStatus(monitors),
		Monitors: make([]PublicStatusMonitor, 0, len(monitors)),
	}
	now := status.clock.Now()
	for _, monitor := range monitors {
		checks, err := status.checks.ListByMonitor(ctx, monitor.ID, 30)
		if err != nil {
			return PublicStatus{}, err
		}
		uptime, err := status.uptime(ctx, monitor.ID, now)
		if err != nil {
			return PublicStatus{}, err
		}
		publicChecks := make([]PublicCheck, 0, len(checks))
		for _, check := range checks {
			publicChecks = append(publicChecks, PublicCheck{Up: check.Up, Latency: check.Latency})
		}
		response.Monitors = append(response.Monitors, PublicStatusMonitor{
			Name: monitor.Name, Type: monitor.Type, Status: monitor.Status,
			LastCheckedAt: monitor.LastCheckedAt, Uptime: uptime, Checks: publicChecks,
		})
	}
	return response, nil
}

func (status *StatusService) uptime(ctx context.Context, monitorID domain.ID, now time.Time) (MonitorUptime, error) {
	periods := []time.Duration{24 * time.Hour, 7 * 24 * time.Hour, 30 * 24 * time.Hour}
	results := make([]UptimeSummary, len(periods))
	for index, period := range periods {
		uptime, err := status.checks.UptimeSince(ctx, monitorID, now.Add(-period))
		if err != nil {
			return MonitorUptime{}, err
		}
		results[index] = summarizeUptime(uptime)
	}
	return MonitorUptime{Last24Hours: results[0], Last7Days: results[1], Last30Days: results[2]}, nil
}

func overallStatus(monitors []domain.Monitor) OverallStatus {
	if len(monitors) == 0 {
		return OverallUnknown
	}
	allUp := true
	for _, monitor := range monitors {
		if monitor.Status == domain.StatusDown {
			return OverallDegraded
		}
		if monitor.Status != domain.StatusUp {
			allUp = false
		}
	}
	if allUp {
		return OverallOperational
	}
	return OverallUnknown
}
