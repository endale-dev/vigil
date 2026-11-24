package scheduler

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/endale-dev/vigil/internal/domain"
	apperrors "github.com/endale-dev/vigil/internal/errors"
	"github.com/endale-dev/vigil/internal/service"
)

type CheckRunner interface {
	Run(ctx context.Context, monitor domain.Monitor, claimToken string) error
}

type Scheduler struct {
	monitors domain.MonitorRepository
	checks   CheckRunner
	clock    service.Clock
	interval time.Duration
	workers  int
	logger   *slog.Logger
}

func New(monitors domain.MonitorRepository, checks CheckRunner, clock service.Clock, interval time.Duration, workers int, logger *slog.Logger) (*Scheduler, error) {
	if monitors == nil || checks == nil || clock == nil || interval <= 0 || workers < 1 || logger == nil {
		return nil, errors.New("scheduler dependencies are incomplete")
	}
	return &Scheduler{monitors: monitors, checks: checks, clock: clock, interval: interval, workers: workers, logger: logger}, nil
}

func (scheduler *Scheduler) Run(ctx context.Context) error {
	ticker := time.NewTicker(scheduler.interval)
	defer ticker.Stop()
	semaphore := make(chan struct{}, scheduler.workers)
	var checks sync.WaitGroup

	for {
		select {
		case <-ctx.Done():
			checks.Wait()
			return nil
		case <-ticker.C:
		}

	claimLoop:
		for {
			if ctx.Err() != nil {
				break claimLoop
			}
			select {
			case semaphore <- struct{}{}:
			default:
				break claimLoop
			}
			claim, err := scheduler.monitors.ClaimDue(ctx, scheduler.clock.Now())
			if err != nil {
				<-semaphore
				if ctx.Err() != nil {
					break claimLoop
				}
				if !errors.Is(err, apperrors.ErrNotFound) {
					scheduler.logger.ErrorContext(ctx, "could not claim due monitor")
				}
				break claimLoop
			}
			if claim == nil {
				<-semaphore
				scheduler.logger.ErrorContext(ctx, "monitor repository returned an empty claim")
				break claimLoop
			}
			checks.Add(1)
			go scheduler.runCheck(ctx, semaphore, claim, &checks)
		}
	}
}

func (scheduler *Scheduler) runCheck(ctx context.Context, semaphore chan struct{}, claim *domain.MonitorClaim, checks *sync.WaitGroup) {
	defer checks.Done()
	defer func() { <-semaphore }()
	checkCtx, cancel := context.WithTimeout(ctx, claim.Monitor.Timeout)
	defer cancel()
	defer func() {
		if recover() != nil {
			scheduler.logger.ErrorContext(ctx, "monitor check panicked")
		}
	}()
	if err := scheduler.checks.Run(checkCtx, claim.Monitor, claim.ClaimToken); err != nil {
		scheduler.logger.ErrorContext(ctx, "monitor check failed")
	}
}
