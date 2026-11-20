package notify

import (
	"context"
	"errors"
	"testing"

	"github.com/endale-dev/vigil/internal/domain"
	"github.com/endale-dev/vigil/internal/service"
)

func TestEmailNotifierSendsDownAndRecoveryMail(t *testing.T) {
	mailer := &recordingMailer{}
	notifier := NewEmailNotifier(mailer)
	user := domain.User{Email: "owner@example.test", Notify: domain.NotifySettings{EmailAlerts: true}}
	if notifier.Name() != domain.ChannelEmail || !notifier.Enabled(user) {
		t.Fatal("email notifier name or enabled state is incorrect")
	}
	monitor := domain.Monitor{Name: "API", Target: "https://example.test/health"}
	for _, test := range []struct {
		status  domain.Status
		subject string
	}{{domain.StatusDown, "[DOWN] API"}, {domain.StatusUp, "[RECOVERED] API"}} {
		alert := service.Alert{Monitor: monitor, Status: test.status, Detail: "HTTP status changed"}
		if err := notifier.Notify(context.Background(), user, alert); err != nil {
			t.Fatalf("Notify(%s) error = %v", test.status, err)
		}
		if got := mailer.messages[len(mailer.messages)-1]; got.To != user.Email || got.Subject != test.subject || got.Body != "https://example.test/health\nHTTP status changed" {
			t.Fatalf("sent mail = %+v, want recipient and subject %q", got, test.subject)
		}
	}
}

func TestEmailNotifierSkipsDisabledAndUnknownAlerts(t *testing.T) {
	mailer := &recordingMailer{}
	notifier := NewEmailNotifier(mailer)
	user := domain.User{Email: "owner@example.test"}
	if notifier.Enabled(user) {
		t.Fatal("notifier enabled when email alerts are disabled")
	}
	if err := notifier.Notify(context.Background(), user, service.Alert{Status: domain.StatusDown}); err != nil {
		t.Fatalf("disabled Notify() error = %v", err)
	}
	user.Notify.EmailAlerts = true
	if err := notifier.Notify(context.Background(), user, service.Alert{Status: domain.StatusUnknown}); err != nil {
		t.Fatalf("unknown alert Notify() error = %v", err)
	}
	if len(mailer.messages) != 0 {
		t.Fatalf("unexpected messages = %+v", mailer.messages)
	}
}

func TestEmailNotifierReturnsMailerError(t *testing.T) {
	want := errors.New("delivery failed")
	mailer := &recordingMailer{err: want}
	notifier := NewEmailNotifier(mailer)
	user := domain.User{Email: "owner@example.test", Notify: domain.NotifySettings{EmailAlerts: true}}
	err := notifier.Notify(context.Background(), user, service.Alert{Monitor: domain.Monitor{Name: "API"}, Status: domain.StatusDown})
	if !errors.Is(err, want) {
		t.Fatalf("Notify() error = %v, want %v", err, want)
	}
}

type recordingMailer struct {
	messages []service.Mail
	err      error
}

func (mailer *recordingMailer) Send(_ context.Context, message service.Mail) error {
	mailer.messages = append(mailer.messages, message)
	return mailer.err
}
