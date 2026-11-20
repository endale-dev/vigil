package service

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/endale-dev/vigil/internal/domain"
)

func TestAlertServiceNotifiesEnabledChannelsWithTimeout(t *testing.T) {
	var logs strings.Builder
	mail := &alertServiceNotifier{channel: domain.ChannelEmail, enabled: true, err: errors.New("provider token secret")}
	telegram := &alertServiceNotifier{channel: domain.ChannelTelegram, enabled: false}
	service, err := NewAlertService([]Notifier{mail, telegram}, slog.New(slog.NewTextHandler(&logs, nil)))
	if err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	alert := Alert{Monitor: domain.Monitor{Name: "API"}, Status: domain.StatusDown, Detail: "connection refused"}
	service.Send(context.Background(), domain.User{ID: "user-1"}, alert)
	if mail.calls != 1 || telegram.calls != 0 || mail.alert != alert {
		t.Fatalf("notifier calls = %d/%d, email alert = %+v", mail.calls, telegram.calls, mail.alert)
	}
	if mail.deadline.IsZero() || mail.deadline.Sub(started) < 14*time.Second || mail.deadline.Sub(started) > 16*time.Second {
		t.Fatalf("notifier deadline = %s, want about 15 seconds from call", mail.deadline)
	}
	if strings.Contains(logs.String(), "provider token secret") || !strings.Contains(logs.String(), "email") {
		t.Fatalf("failure log exposed provider error or omitted channel: %s", logs.String())
	}
}

func TestNewAlertServiceRejectsDuplicateChannels(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&strings.Builder{}, nil))
	first := &alertServiceNotifier{channel: domain.ChannelEmail}
	second := &alertServiceNotifier{channel: domain.ChannelEmail}
	if service, err := NewAlertService([]Notifier{first, second}, logger); err == nil || service != nil {
		t.Fatalf("NewAlertService() = (%v, %v), want duplicate-channel error", service, err)
	}
}

type alertServiceNotifier struct {
	channel  domain.Channel
	enabled  bool
	calls    int
	alert    Alert
	deadline time.Time
	err      error
}

func (notifier *alertServiceNotifier) Name() domain.Channel { return notifier.channel }

func (notifier *alertServiceNotifier) Enabled(domain.User) bool { return notifier.enabled }

func (notifier *alertServiceNotifier) Notify(ctx context.Context, _ domain.User, alert Alert) error {
	notifier.calls++
	notifier.alert = alert
	notifier.deadline, _ = ctx.Deadline()
	return notifier.err
}
