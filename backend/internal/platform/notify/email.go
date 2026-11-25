package notify

import (
	"context"
	"fmt"

	"github.com/endale-dev/vigil/internal/domain"
	"github.com/endale-dev/vigil/internal/service"
)

type EmailNotifier struct {
	mailer service.Mailer
}

func NewEmailNotifier(mailer service.Mailer) *EmailNotifier {
	return &EmailNotifier{mailer: mailer}
}

func (*EmailNotifier) Name() domain.Channel {
	return domain.ChannelEmail
}

func (*EmailNotifier) Enabled(user domain.User) bool {
	return user.Notify.EmailAlerts
}

func (notifier *EmailNotifier) Notify(ctx context.Context, user domain.User, alert service.Alert) error {
	if !notifier.Enabled(user) {
		return nil
	}
	var subject string
	switch alert.Status {
	case domain.StatusDown:
		subject = "[DOWN] " + alert.Monitor.Name
	case domain.StatusUp:
		subject = "[RECOVERED] " + alert.Monitor.Name
	default:
		return nil
	}
	destination := user.Notify.EmailDestination
	if destination == "" {
		destination = user.Email
	}
	return notifier.mailer.Send(ctx, service.Mail{
		To:      destination,
		Subject: subject,
		Body:    fmt.Sprintf("%s\n%s", alert.Monitor.Target, alert.Detail),
	})
}

var _ service.Notifier = (*EmailNotifier)(nil)
