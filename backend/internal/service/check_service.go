package service

import (
	"context"
	"errors"
	"log/slog"

	"github.com/endale-dev/vigil/internal/domain"
)

type CheckService struct {
	monitors         domain.MonitorRepository
	checks           domain.CheckRepository
	prober           Prober
	clock            Clock
	failureThreshold int
	logger           *slog.Logger
}

func NewCheckService(monitors domain.MonitorRepository, checks domain.CheckRepository, prober Prober, clock Clock, failureThreshold int, logger *slog.Logger) (*CheckService, error) {
	if monitors == nil || checks == nil || prober == nil || clock == nil || failureThreshold < 1 || logger == nil {
		return nil, errors.New("check service dependencies are incomplete")
	}
	return &CheckService{
		monitors: monitors, checks: checks, prober: prober, clock: clock,
		failureThreshold: failureThreshold, logger: logger,
	}, nil
}

func (checks *CheckService) Run(ctx context.Context, monitor domain.Monitor, claimToken string) error {
	result := checks.prober.Probe(ctx, monitor)
	checkedAt := checks.clock.Now()
	if err := checks.checks.Insert(ctx, &domain.Check{
		MonitorID: monitor.ID,
		CheckedAt: checkedAt,
		Up:        result.Up,
		Latency:   result.Latency,
		Detail:    result.Detail,
	}); err != nil {
		checks.logger.ErrorContext(ctx, "monitor check could not be saved")
		return err
	}
	status, failures := transitionMonitor(monitor, result, checks.failureThreshold)
	if err := checks.monitors.RecordResult(ctx, monitor.ID, claimToken, status, failures, checkedAt); err != nil {
		checks.logger.ErrorContext(ctx, "monitor result could not be saved")
		return err
	}
	return nil
}
