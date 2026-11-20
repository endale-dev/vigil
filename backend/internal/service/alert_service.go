package service

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/endale-dev/vigil/internal/domain"
)

type AlertService struct {
	notifiers []Notifier
	logger    *slog.Logger
}

func NewAlertService(notifiers []Notifier, logger *slog.Logger) (*AlertService, error) {
	if logger == nil {
		return nil, errors.New("alert service logger is required")
	}
	seen := make(map[domain.Channel]struct{}, len(notifiers))
	for _, notifier := range notifiers {
		if notifier == nil {
			return nil, errors.New("alert service notifier is nil")
		}
		if _, exists := seen[notifier.Name()]; exists {
			return nil, errors.New("alert service notifier channel is duplicated")
		}
		seen[notifier.Name()] = struct{}{}
	}
	return &AlertService{notifiers: append([]Notifier(nil), notifiers...), logger: logger}, nil
}

func (alerts *AlertService) Send(ctx context.Context, owner domain.User, alert Alert) {
	for _, notifier := range alerts.notifiers {
		if !notifier.Enabled(owner) {
			continue
		}
		notifyCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		err := notifier.Notify(notifyCtx, owner, alert)
		cancel()
		if err != nil {
			alerts.logger.ErrorContext(ctx, "alert notification failed", "channel", notifier.Name())
		}
	}
}

var _ Alerter = (*AlertService)(nil)
