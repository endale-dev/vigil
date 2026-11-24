package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/endale-dev/vigil/internal/config"
	"github.com/endale-dev/vigil/internal/logging"
	"github.com/endale-dev/vigil/internal/platform/clock"
	"github.com/endale-dev/vigil/internal/platform/mail"
	"github.com/endale-dev/vigil/internal/platform/notify"
	"github.com/endale-dev/vigil/internal/platform/probe"
	"github.com/endale-dev/vigil/internal/repository/mongo"
	"github.com/endale-dev/vigil/internal/scheduler"
	"github.com/endale-dev/vigil/internal/server"
	"github.com/endale-dev/vigil/internal/service"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err == nil {
		logger := logging.New(cfg.AppEnv, os.Stderr)
		err = run(ctx, cfg, logger)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, cfg config.Config, logger *slog.Logger) error {
	client, err := mongo.Connect(ctx, cfg.MongoURI, cfg.MongoDB)
	if err != nil {
		return err
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := client.Close(closeCtx); err != nil {
			logger.Error("MongoDB disconnect failed")
		}
	}()

	database := client.Database()
	if err := mongo.CreateIndexes(ctx, database, cfg.CheckRetention); err != nil {
		return err
	}

	users := mongo.NewUserRepository(database)
	monitors := mongo.NewMonitorRepository(database)
	checks := mongo.NewCheckRepository(database)
	incidents := mongo.NewIncidentRepository(database)
	appClock := clock.System{}
	prober := probe.New(cfg.AllowPrivateTargets, appClock)
	mailer := service.Mailer(mail.NewLogMailer(logger))
	if cfg.SMTPHost != "" {
		mailer = mail.NewSMTPMailer(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPFrom)
	}
	alerter, err := service.NewAlertService([]service.Notifier{notify.NewEmailNotifier(mailer)}, logger)
	if err != nil {
		return err
	}
	checker, err := service.NewCheckService(monitors, checks, incidents, users, prober, alerter, appClock, cfg.FailureThreshold, logger)
	if err != nil {
		return err
	}
	monitorScheduler, err := scheduler.New(monitors, checker, appClock, cfg.SchedulerTick, cfg.CheckWorkers, logger)
	if err != nil {
		return err
	}

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	results := make(chan error, 2)
	go func() { results <- server.Run(runCtx, cfg.AppAddr, logger) }()
	go func() { results <- monitorScheduler.Run(runCtx) }()

	firstErr := <-results
	cancel()
	secondErr := <-results
	if firstErr != nil && !errors.Is(firstErr, context.Canceled) {
		return firstErr
	}
	if secondErr != nil && !errors.Is(secondErr, context.Canceled) {
		return secondErr
	}
	return nil
}
