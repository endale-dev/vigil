package main

import (
	"context"
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
