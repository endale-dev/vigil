package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/endale-dev/vigil/backend/internal/config"
	"github.com/endale-dev/vigil/backend/internal/server"
)

func Run(ctx context.Context, cfg config.Config, logger *slog.Logger) error {
	listener, err := net.Listen("tcp", cfg.AppAddr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	httpServer := server.New(cfg.AppAddr)
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
			return fmt.Errorf("shutdown HTTP server: %w", err)
		}
		if err := <-serveErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve HTTP: %w", err)
		}
		return nil
	}
}
