package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/endale-dev/vigil/internal/config"
	"github.com/endale-dev/vigil/internal/handler"
	"github.com/endale-dev/vigil/internal/logging"
	"github.com/endale-dev/vigil/internal/platform/clock"
	"github.com/endale-dev/vigil/internal/platform/google"
	"github.com/endale-dev/vigil/internal/platform/mail"
	"github.com/endale-dev/vigil/internal/platform/notify"
	"github.com/endale-dev/vigil/internal/platform/probe"
	"github.com/endale-dev/vigil/internal/platform/security"
	"github.com/endale-dev/vigil/internal/repository/mongo"
	"github.com/endale-dev/vigil/internal/routes"
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
	clientClosed := false
	closeClient := func() {
		if clientClosed {
			return
		}
		clientClosed = true
		closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := client.Close(closeCtx); err != nil {
			logger.Error("MongoDB disconnect failed")
		}
	}
	defer closeClient()

	database := client.Database()
	if err := mongo.CreateIndexes(ctx, database, cfg.CheckRetention); err != nil {
		return err
	}

	dependencies, err := buildDependencies(cfg, client, logger)
	if err != nil {
		return err
	}
	handlers, err := buildHandlers(cfg, client, dependencies, logger)
	if err != nil {
		return err
	}

	return runLifecycle(ctx,
		func(ctx context.Context, onStopAccepting func()) error {
			return server.RunWithRoutes(
				ctx, cfg.AppAddr, cfg.AppEnv, embeddedFrontend, handlers, server.RequireAuth(dependencies.authentication),
				cfg.TrustProxyHeaders, logger, onStopAccepting,
			)
		},
		dependencies.scheduler.Run,
		closeClient,
	)
}

func runLifecycle(ctx context.Context, runHTTP func(context.Context, func()) error, runScheduler func(context.Context) error, closeDependencies func()) error {
	err := runProcesses(ctx, runHTTP, runScheduler)
	closeDependencies()
	return err
}

func runProcesses(ctx context.Context, runHTTP func(context.Context, func()) error, runScheduler func(context.Context) error) error {
	httpCtx, stopHTTP := context.WithCancel(context.WithoutCancel(ctx))
	defer stopHTTP()
	schedulerCtx, stopScheduler := context.WithCancel(context.WithoutCancel(ctx))
	defer stopScheduler()
	var stopSchedulerOnce sync.Once
	stopSchedulerAfterHTTP := func() { stopSchedulerOnce.Do(stopScheduler) }

	httpDone := make(chan error, 1)
	schedulerDone := make(chan error, 1)
	go func() {
		err := runHTTP(httpCtx, stopSchedulerAfterHTTP)
		stopSchedulerAfterHTTP()
		httpDone <- err
	}()
	go func() { schedulerDone <- runScheduler(schedulerCtx) }()

	select {
	case <-ctx.Done():
		stopHTTP()
		httpErr := <-httpDone
		return processErrors(httpErr, <-schedulerDone)
	case httpErr := <-httpDone:
		stopHTTP()
		stopSchedulerAfterHTTP()
		return processErrors(httpErr, <-schedulerDone)
	case schedulerErr := <-schedulerDone:
		stopHTTP()
		stopSchedulerAfterHTTP()
		return processErrors(schedulerErr, <-httpDone)
	}
}

func processErrors(first, second error) error {
	if errors.Is(first, context.Canceled) {
		first = nil
	}
	if errors.Is(second, context.Canceled) {
		second = nil
	}
	return errors.Join(first, second)
}

type applicationDependencies struct {
	authentication *service.AuthenticationService
	account        *service.AccountService
	monitors       *service.MonitorService
	status         *service.StatusService
	scheduler      *scheduler.Scheduler
}

func buildDependencies(cfg config.Config, client *mongo.Client, logger *slog.Logger) (applicationDependencies, error) {
	database := client.Database()
	users := mongo.NewUserRepository(database)
	sessions := mongo.NewSessionRepository(database)
	passwordResets := mongo.NewPasswordResetRepository(database)
	monitors := mongo.NewMonitorRepository(database)
	checks := mongo.NewCheckRepository(database)
	incidents := mongo.NewIncidentRepository(database)
	appClock := clock.System{}
	mailer := service.Mailer(mail.NewLogMailer(logger))
	if cfg.SMTPHost != "" {
		mailer = mail.NewSMTPMailer(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPFrom)
	}
	notifiers := newNotifiers(cfg, mailer)
	alerter, err := service.NewAlertService(notifiers, logger)
	if err != nil {
		return applicationDependencies{}, err
	}
	googleProvider := newGoogleProvider(cfg)
	authentication, err := service.NewAuthenticationService(service.AuthenticationDependencies{
		Users: users, Sessions: sessions, PasswordResets: passwordResets,
		Passwords: security.NewPasswordHasher(), Tokens: security.NewTokenGenerator(),
		Mailer: mailer, Google: googleProvider, Clock: appClock,
		BaseURL: cfg.AppBaseURL, SessionTTL: cfg.SessionTTL, Logger: logger,
	})
	if err != nil {
		return applicationDependencies{}, err
	}
	account, err := service.NewAccountService(users, cfg.TelegramBotToken != "", notifiers...)
	if err != nil {
		return applicationDependencies{}, err
	}
	monitorService, err := service.NewMonitorService(monitors, checks, incidents, appClock, cfg.MaxMonitorsPerUser)
	if err != nil {
		return applicationDependencies{}, err
	}
	status, err := service.NewStatusService(users, monitors, checks, appClock)
	if err != nil {
		return applicationDependencies{}, err
	}
	checker, err := service.NewCheckService(monitors, checks, incidents, users, probe.New(cfg.AllowPrivateTargets, appClock), alerter, appClock, cfg.FailureThreshold, logger)
	if err != nil {
		return applicationDependencies{}, err
	}
	monitorScheduler, err := scheduler.New(monitors, checker, appClock, cfg.SchedulerTick, cfg.CheckWorkers, logger)
	if err != nil {
		return applicationDependencies{}, err
	}
	return applicationDependencies{
		authentication: authentication,
		account:        account,
		monitors:       monitorService,
		status:         status,
		scheduler:      monitorScheduler,
	}, nil
}

func buildHandlers(cfg config.Config, client *mongo.Client, dependencies applicationDependencies, logger *slog.Logger) (routes.Handlers, error) {
	capabilities := handler.CapabilitiesDTO{Google: cfg.GoogleClientID != "", Telegram: cfg.TelegramBotToken != ""}
	authentication, err := handler.NewAuthenticationHandler(dependencies.authentication, cfg.CookieSecure, capabilities, logger)
	if err != nil {
		return routes.Handlers{}, err
	}
	account, err := handler.NewAccountHandler(dependencies.authentication, dependencies.account, capabilities, logger)
	if err != nil {
		return routes.Handlers{}, err
	}
	monitors, err := handler.NewMonitorHandler(dependencies.authentication, dependencies.monitors, logger)
	if err != nil {
		return routes.Handlers{}, err
	}
	publicStatus, err := handler.NewPublicStatusHandler(dependencies.status, logger)
	if err != nil {
		return routes.Handlers{}, err
	}
	health, err := handler.NewHealthHandler(client)
	if err != nil {
		return routes.Handlers{}, err
	}
	return routes.Handlers{
		Authentication: authentication,
		Account:        account,
		Monitors:       monitors,
		PublicStatus:   publicStatus,
		Health:         health,
	}, nil
}

func newNotifiers(cfg config.Config, mailer service.Mailer) []service.Notifier {
	notifiers := []service.Notifier{notify.NewEmailNotifier(mailer)}
	if cfg.TelegramBotToken != "" {
		notifiers = append(notifiers, notify.NewTelegramNotifier(cfg.TelegramBotToken))
	}
	return notifiers
}

func newGoogleProvider(cfg config.Config) service.GoogleProvider {
	if cfg.GoogleClientID == "" {
		return nil
	}
	return google.NewProvider(cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.AppBaseURL+"/api/auth/google/callback")
}
