package main

import (
	"context"
	"net/url"
	"testing"

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
