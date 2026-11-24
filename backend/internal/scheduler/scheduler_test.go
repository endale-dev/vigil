package scheduler

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/endale-dev/vigil/internal/domain"
	apperrors "github.com/endale-dev/vigil/internal/errors"
	"github.com/endale-dev/vigil/internal/service"
)

func TestNewRequiresCompleteDependencies(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	repository := &claimRepository{}
	checks := &blockingRunner{}
	clock := &fixedClock{now: time.Now()}
	for _, test := range []struct {
		name     string
		repo     domain.MonitorRepository
		checks   CheckRunner
		clock    service.Clock
		interval time.Duration
		workers  int
	}{
		{name: "missing repository", checks: checks, clock: clock, interval: time.Second, workers: 1},
		{name: "missing checker", repo: repository, clock: clock, interval: time.Second, workers: 1},
		{name: "missing clock", repo: repository, checks: checks, interval: time.Second, workers: 1},
		{name: "invalid interval", repo: repository, checks: checks, clock: clock, workers: 1},
		{name: "invalid workers", repo: repository, checks: checks, clock: clock, interval: time.Second},
	} {
		t.Run(test.name, func(t *testing.T) {
			if scheduler, err := New(test.repo, test.checks, test.clock, test.interval, test.workers, logger); err == nil || scheduler != nil {
				t.Fatalf("New() = (%v, %v), want dependency error", scheduler, err)
			}
		})
	}
}

func TestSchedulerClaimsOnlyWhenAWorkerIsAvailable(t *testing.T) {
	now := time.Date(2027, time.February, 15, 16, 17, 18, 0, time.UTC)
	repository := &claimRepository{claims: []*domain.MonitorClaim{
		{Monitor: domain.Monitor{ID: "monitor-1", Timeout: time.Minute}, ClaimToken: "claim-1"},
		{Monitor: domain.Monitor{ID: "monitor-2", Timeout: 2 * time.Minute}, ClaimToken: "claim-2"},
	}}
	runner := &blockingRunner{started: make(chan runObservation, 2), release: make(chan struct{}), finished: make(chan struct{}, 2)}
	scheduler, err := New(repository, runner, &fixedClock{now: now}, 5*time.Millisecond, 2, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- scheduler.Run(ctx) }()

	observations := []runObservation{receiveRun(t, runner.started), receiveRun(t, runner.started)}
	if repository.calls != 2 {
		t.Fatalf("ClaimDue calls while both workers are busy = %d, want 2", repository.calls)
	}
	for _, observation := range observations {
		if !observation.deadlineSet || observation.remaining <= 0 || observation.remaining > observation.timeout {
			t.Errorf("check timeout = %s, want no more than %s", observation.remaining, observation.timeout)
		}
	}
	if !repository.claimTimes[0].Equal(now) || !repository.claimTimes[1].Equal(now) {
		t.Fatalf("claim times = %v, want injected time %s", repository.claimTimes, now)
	}
	close(runner.release)
	for range observations {
		select {
		case <-runner.finished:
		case <-time.After(time.Second):
			t.Fatal("check did not finish")
		}
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("scheduler did not stop")
	}
}

func TestSchedulerStopsClaimingAfterNoDueMonitor(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	repository := &claimRepository{onEmpty: cancel}
	runner := &blockingRunner{started: make(chan runObservation, 1), release: make(chan struct{}), finished: make(chan struct{}, 1)}
	scheduler, err := New(repository, runner, &fixedClock{now: time.Now()}, time.Millisecond, 1, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	if err := scheduler.Run(ctx); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if repository.calls != 1 || len(runner.started) != 0 {
		t.Fatalf("empty claim calls/checks = %d/%d, want 1/0", repository.calls, len(runner.started))
	}
}

func TestSchedulerWaitsForChecksWhenCanceled(t *testing.T) {
	repository := &claimRepository{claims: []*domain.MonitorClaim{
		{Monitor: domain.Monitor{ID: "monitor-1", Timeout: time.Minute}, ClaimToken: "claim-1"},
		{Monitor: domain.Monitor{ID: "monitor-2", Timeout: time.Minute}, ClaimToken: "claim-2"},
	}}
	runner := &nonCooperativeRunner{started: make(chan struct{}, 1), release: make(chan struct{}), finished: make(chan struct{}, 1)}
	scheduler, err := New(repository, runner, &fixedClock{now: time.Now()}, 5*time.Millisecond, 1, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- scheduler.Run(ctx) }()
	select {
	case <-runner.started:
	case <-time.After(time.Second):
		t.Fatal("scheduler did not start the check")
	}
	cancel()
	select {
	case err := <-done:
		t.Fatalf("Run() returned before the check drained: %v", err)
	case <-time.After(25 * time.Millisecond):
	}
	close(runner.release)
	select {
	case <-runner.finished:
	case <-time.After(time.Second):
		t.Fatal("check did not finish")
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Run() did not return after checks drained")
	}
	if repository.calls != 1 {
		t.Fatalf("claims after cancellation = %d, want 1", repository.calls)
	}
}

func TestSchedulerRecoversFromWorkerPanic(t *testing.T) {
	var logs strings.Builder
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	repository := &claimRepository{claims: []*domain.MonitorClaim{
		{Monitor: domain.Monitor{ID: "monitor-1", Timeout: time.Minute}, ClaimToken: "claim-1"},
		{Monitor: domain.Monitor{ID: "monitor-2", Timeout: time.Minute}, ClaimToken: "claim-2"},
	}}
	runner := &panicOnceRunner{started: make(chan struct{}, 1), release: make(chan struct{}), finished: make(chan struct{}, 1)}
	scheduler, err := New(repository, runner, &fixedClock{now: time.Now()}, 5*time.Millisecond, 1, logger)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- scheduler.Run(ctx) }()
	select {
	case <-runner.started:
	case <-time.After(time.Second):
		t.Fatal("scheduler did not run the monitor after the panic")
	}
	if strings.Contains(logs.String(), "private panic detail") || !strings.Contains(logs.String(), "monitor check panicked") {
		t.Fatalf("panic log = %s", logs.String())
	}
	close(runner.release)
	select {
	case <-runner.finished:
	case <-time.After(time.Second):
		t.Fatal("check did not finish")
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("scheduler did not stop")
	}
}

type claimRepository struct {
	domain.MonitorRepository
	claims     []*domain.MonitorClaim
	claimTimes []time.Time
	calls      int
	onEmpty    context.CancelFunc
}

func (repository *claimRepository) ClaimDue(_ context.Context, now time.Time) (*domain.MonitorClaim, error) {
	repository.calls++
	repository.claimTimes = append(repository.claimTimes, now)
	if len(repository.claims) == 0 {
		if repository.onEmpty != nil {
			repository.onEmpty()
		}
		return nil, apperrors.ErrNotFound
	}
	claim := repository.claims[0]
	repository.claims = repository.claims[1:]
	return claim, nil
}

type fixedClock struct{ now time.Time }

func (clock *fixedClock) Now() time.Time { return clock.now }

type runObservation struct {
	timeout     time.Duration
	remaining   time.Duration
	deadlineSet bool
}

type blockingRunner struct {
	started  chan runObservation
	release  chan struct{}
	finished chan struct{}
}

type nonCooperativeRunner struct {
	started  chan struct{}
	release  chan struct{}
	finished chan struct{}
}

func (runner *nonCooperativeRunner) Run(context.Context, domain.Monitor, string) error {
	runner.started <- struct{}{}
	<-runner.release
	runner.finished <- struct{}{}
	return nil
}

type panicOnceRunner struct {
	calls    int
	started  chan struct{}
	release  chan struct{}
	finished chan struct{}
}

func (runner *panicOnceRunner) Run(context.Context, domain.Monitor, string) error {
	runner.calls++
	if runner.calls == 1 {
		panic("private panic detail")
	}
	runner.started <- struct{}{}
	<-runner.release
	runner.finished <- struct{}{}
	return nil
}

func (runner *blockingRunner) Run(ctx context.Context, monitor domain.Monitor, _ string) error {
	deadline, ok := ctx.Deadline()
	runner.started <- runObservation{timeout: monitor.Timeout, remaining: time.Until(deadline), deadlineSet: ok}
	select {
	case <-runner.release:
		runner.finished <- struct{}{}
		return nil
	case <-ctx.Done():
		runner.finished <- struct{}{}
		return ctx.Err()
	}
}

func receiveRun(t *testing.T, runs <-chan runObservation) runObservation {
	t.Helper()
	select {
	case observation := <-runs:
		return observation
	case <-time.After(time.Second):
		t.Fatal("scheduler did not start a check")
		return runObservation{}
	}
}

var _ CheckRunner = (*blockingRunner)(nil)
var _ domain.MonitorRepository = (*claimRepository)(nil)
