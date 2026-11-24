package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"
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
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	httpServer := New(addr)
	if onStopAccepting != nil {
		httpServer.RegisterOnShutdown(onStopAccepting)
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
