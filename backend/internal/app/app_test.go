package app

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/endale-dev/vigil/backend/internal/config"
)

func TestRunStopsAfterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var logs bytes.Buffer
	result := make(chan error, 1)
	go func() {
		result <- Run(ctx, config.Config{AppAddr: "127.0.0.1:0"}, slog.New(slog.NewTextHandler(&logs, nil)))
	}()
	cancel()

	select {
	case err := <-result:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not stop after cancellation")
	}
	if logs.Len() == 0 {
		t.Fatal("Run did not report successful startup")
	}
}

func TestRunReportsListenFailure(t *testing.T) {
	err := Run(context.Background(), config.Config{AppAddr: "not-an-address"}, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	if err == nil {
		t.Fatal("expected listen error")
	}
}
