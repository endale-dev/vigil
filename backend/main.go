package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/endale-dev/vigil/internal/config"
	"github.com/endale-dev/vigil/internal/logging"
	"github.com/endale-dev/vigil/internal/server"
)

func main() {
	err := func() error {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		cfg, err := config.Load()
		if err != nil {
			return err
		}
		logger := logging.New(cfg.AppEnv, os.Stderr)
		return server.Run(ctx, cfg.AppAddr, logger)
	}()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
