package server

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/endale-dev/vigil/internal/routes"
)

func TestRunStopsAfterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var logs bytes.Buffer
	result := make(chan error, 1)
	go func() {
		result <- Run(ctx, "127.0.0.1:0", slog.New(slog.NewTextHandler(&logs, nil)))
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
	err := Run(context.Background(), "not-an-address", slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	if err == nil {
		t.Fatal("expected listen error")
	}
}

func TestRunWithRoutesRequiresProductionFrontendBeforeListening(t *testing.T) {
	err := RunWithRoutes(context.Background(), "not-an-address", "production", nil, routes.Handlers{}, nil, false, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)), nil)
	if err == nil || err.Error() != missingSPAError {
		t.Fatalf("RunWithRoutes() error = %v, want missing frontend build error", err)
	}
}

func TestNewUsesConfiguredHTTPTimeouts(t *testing.T) {
	httpServer := New(":8080")
	if httpServer.ReadHeaderTimeout != 5*time.Second || httpServer.ReadTimeout != 15*time.Second || httpServer.WriteTimeout != 30*time.Second || httpServer.IdleTimeout != 60*time.Second || httpServer.MaxHeaderBytes != 1<<16 {
		t.Fatalf("server timeouts/headers = %s/%s/%s/%s/%d", httpServer.ReadHeaderTimeout, httpServer.ReadTimeout, httpServer.WriteTimeout, httpServer.IdleTimeout, httpServer.MaxHeaderBytes)
	}
}

func TestRunWithHandlerDrainsRequestsBeforeReturning(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}

	started := make(chan struct{})
	release := make(chan struct{})
	releaseRequest := func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}
	defer releaseRequest()
	stoppedAccepting := make(chan struct{})
	serverDone := make(chan error, 1)
	requestDone := make(chan error, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	handler := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		close(started)
		<-release
		writer.WriteHeader(http.StatusNoContent)
	})
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	go func() {
		serverDone <- runWithHandler(ctx, address, handler, logger, func() { close(stoppedAccepting) }, nil)
	}()
	go func() {
		client := http.Client{
			Timeout:   5 * time.Second,
			Transport: &http.Transport{Proxy: nil},
		}
		response, err := client.Get("http://" + address + "/active")
		if err == nil {
			response.Body.Close()
			if response.StatusCode != http.StatusNoContent {
				err = errors.New("active request returned an unexpected status")
			}
		}
		requestDone <- err
	}()

	select {
	case <-started:
	case err := <-serverDone:
		t.Fatalf("server stopped before the in-flight request started: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("server did not start the in-flight request")
	}
	cancel()
	select {
	case <-stoppedAccepting:
	case <-time.After(2 * time.Second):
		t.Fatal("server did not signal that it stopped accepting requests")
	}
	select {
	case err := <-serverDone:
		t.Fatalf("server returned before the in-flight request drained: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	releaseRequest()
	select {
	case err := <-requestDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("in-flight request did not complete")
	}
	select {
	case err := <-serverDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server did not return after draining the request")
	}
}
