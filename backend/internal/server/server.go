package server

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/endale-dev/vigil/internal/routes"
)

func New(addr string) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           http.NewServeMux(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 16,
	}
}

func Run(ctx context.Context, addr string, logger *slog.Logger) error {
	return RunWithShutdown(ctx, addr, logger, nil)
}

func RunWithShutdown(ctx context.Context, addr string, logger *slog.Logger, onStopAccepting func()) error {
	return runWithHandler(ctx, addr, New(addr).Handler, logger, onStopAccepting, nil)
}

func RunWithRoutes(ctx context.Context, addr string, appEnv string, embeddedFrontend fs.FS, handlers routes.Handlers, requireAuth func(http.Handler) http.Handler, trustProxyHeaders bool, logger *slog.Logger, onStopAccepting func()) error {
	assets, err := embeddedSPA(appEnv, embeddedFrontend)
	if err != nil {
		return err
	}
	mux, err := routes.New(handlers, requireAuth)
	if err != nil {
		return err
	}
	mux.Handle("/", NewSPAHandler(assets))
	limiter := NewRateLimiter(trustProxyHeaders)
	requestHandler := RequestLog(logger)(Recover(logger)(SecurityHeaders(CSRF(RateLimit(limiter)(mux)))))
	cleanupCtx, stopCleanup := context.WithCancel(ctx)
	defer stopCleanup()
	cleanupDone := make(chan struct{})
	cleanupStarted := false
	err = runWithHandler(ctx, addr, requestHandler, logger, onStopAccepting, func() {
		cleanupStarted = true
		go func() {
			defer close(cleanupDone)
			limiter.RunCleanup(cleanupCtx)
		}()
	})
	stopCleanup()
	if cleanupStarted {
		<-cleanupDone
	}
	return err
}

func runWithHandler(ctx context.Context, addr string, handler http.Handler, logger *slog.Logger, onStopAccepting func(), onStart func()) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	httpServer := New(addr)
	httpServer.Handler = handler
	if onStopAccepting != nil {
		httpServer.RegisterOnShutdown(onStopAccepting)
	}
	if onStart != nil {
		onStart()
	}
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- httpServer.Serve(listener)
	}()
	logger.Info("HTTP server started", "address", listener.Addr().String())

	select {
	case err := <-serveErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve HTTP: %w", err)
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			shutdownErr := fmt.Errorf("shutdown HTTP server: %w", err)
			if closeErr := httpServer.Close(); closeErr != nil {
				shutdownErr = errors.Join(shutdownErr, fmt.Errorf("close HTTP server: %w", closeErr))
			}
			if serveErr := <-serveErr; serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
				shutdownErr = errors.Join(shutdownErr, fmt.Errorf("serve HTTP: %w", serveErr))
			}
			return shutdownErr
		}
		if err := <-serveErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve HTTP: %w", err)
		}
		return nil
	}
}
