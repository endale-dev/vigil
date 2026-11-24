package main

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/endale-dev/vigil/internal/config"
	"github.com/endale-dev/vigil/internal/domain"
	"github.com/endale-dev/vigil/internal/service"
)

type testMailer struct{}

func (testMailer) Send(context.Context, service.Mail) error { return nil }

func TestNewNotifiers(t *testing.T) {
	for _, test := range []struct {
		name       string
		token      string
		wantLength int
	}{
		{name: "email only", wantLength: 1},
		{name: "telegram configured", token: "bot-token", wantLength: 2},
	} {
		t.Run(test.name, func(t *testing.T) {
			notifiers := newNotifiers(config.Config{TelegramBotToken: test.token}, testMailer{})
			if len(notifiers) != test.wantLength {
				t.Fatalf("got %d notifiers, want %d", len(notifiers), test.wantLength)
			}
			if notifiers[0].Name() != domain.ChannelEmail {
				t.Fatalf("first notifier is %q, want email", notifiers[0].Name())
			}
			if test.token != "" && notifiers[1].Name() != domain.ChannelTelegram {
				t.Fatalf("second notifier is %q, want telegram", notifiers[1].Name())
			}
		})
	}
}

func TestNewGoogleProvider(t *testing.T) {
	if provider := newGoogleProvider(config.Config{}); provider != nil {
		t.Fatal("Google provider should be disabled without credentials")
	}

	provider := newGoogleProvider(config.Config{
		GoogleClientID:     "client-id",
		GoogleClientSecret: "client-secret",
		AppBaseURL:         "https://vigil.example",
	})
	if provider == nil {
		t.Fatal("Google provider should be enabled with credentials")
	}
	authorizationURL, err := url.Parse(provider.AuthURL("state", "challenge"))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := authorizationURL.Query().Get("redirect_uri"), "https://vigil.example/api/auth/google/callback"; got != want {
		t.Fatalf("got redirect URI %q, want %q", got, want)
	}
}

func TestRunLifecycleDrainsBeforeClosingDependencies(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	httpStarted := make(chan struct{})
	schedulerStarted := make(chan struct{})
	httpStoppedAccepting := make(chan struct{})
	schedulerStoppedClaiming := make(chan struct{})
	allowHTTPDrain := make(chan struct{})
	allowCheckDrain := make(chan struct{})
	httpDrained := make(chan struct{})
	checksDrained := make(chan struct{})
	dependenciesClosed := make(chan struct{})
	result := make(chan error, 1)

	runHTTP := func(ctx context.Context, onStopAccepting func()) error {
		close(httpStarted)
		<-ctx.Done()
		close(httpStoppedAccepting)
		onStopAccepting()
		<-allowHTTPDrain
		close(httpDrained)
		return nil
	}
	runScheduler := func(ctx context.Context) error {
		close(schedulerStarted)
		<-ctx.Done()
		select {
		case <-httpStoppedAccepting:
		default:
			t.Error("scheduler stopped claiming before HTTP stopped accepting")
		}
		close(schedulerStoppedClaiming)
		<-allowCheckDrain
		close(checksDrained)
		return nil
	}
	go func() {
		result <- runLifecycle(ctx, runHTTP, runScheduler, func() { close(dependenciesClosed) })
	}()
	waitForSignal(t, httpStarted, "HTTP startup")
	waitForSignal(t, schedulerStarted, "scheduler startup")
	cancel()
	waitForSignal(t, httpStoppedAccepting, "HTTP stop accepting")
	waitForSignal(t, schedulerStoppedClaiming, "scheduler stop claiming")

	assertNotSignaled(t, dependenciesClosed, "dependencies closed before checks drained")
	close(allowCheckDrain)
	waitForSignal(t, checksDrained, "active check drain")
	assertNotSignaled(t, dependenciesClosed, "dependencies closed before HTTP drained")
	close(allowHTTPDrain)
	waitForSignal(t, httpDrained, "HTTP drain")
	waitForSignal(t, dependenciesClosed, "dependency cleanup")

	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("lifecycle returned an error after cancellation: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("lifecycle did not return after shutdown completed")
	}
}

func waitForSignal(t *testing.T, signal <-chan struct{}, name string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for %s", name)
	}
}

func assertNotSignaled(t *testing.T, signal <-chan struct{}, message string) {
	t.Helper()
	select {
	case <-signal:
		t.Fatal(message)
	default:
	}
}
