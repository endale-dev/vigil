package service

import (
	"context"
	"errors"

	"github.com/endale-dev/vigil/internal/domain"
	apperrors "github.com/endale-dev/vigil/internal/errors"
)

type AccountService struct {
	users           domain.UserRepository
	telegramEnabled bool
	notifiers       map[domain.Channel]Notifier
}

func NewAccountService(users domain.UserRepository, telegramEnabled bool, notifiers ...Notifier) (*AccountService, error) {
	if users == nil {
		return nil, errors.New("account service user repository is required")
	}
	account := &AccountService{
		users:           users,
		telegramEnabled: telegramEnabled,
		notifiers:       make(map[domain.Channel]Notifier, len(notifiers)),
	}
	for _, notifier := range notifiers {
		if notifier == nil || account.notifiers[notifier.Name()] != nil {
			return nil, errors.New("account service notifiers are invalid")
		}
		account.notifiers[notifier.Name()] = notifier
	}
	return account, nil
}

func (account *AccountService) UpdateNotifications(ctx context.Context, userID domain.ID, notify domain.NotifySettings) (*domain.User, error) {
	if notify.TelegramChatID != "" {
		if !account.telegramEnabled {
			return nil, apperrors.ErrProviderDisabled
		}
		if !validTelegramChatID(notify.TelegramChatID) {
			return nil, apperrors.ValidationError{Field: "telegramChatId", Message: "must be empty or a 5 to 20 digit chat ID"}
		}
	}
	user, err := account.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	user.Notify = notify
	if err := account.users.Update(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func validTelegramChatID(chatID string) bool {
	digits := chatID
	if len(digits) > 0 && digits[0] == '-' {
		digits = digits[1:]
	}
	if len(digits) < 5 || len(digits) > 20 {
		return false
	}
	for _, digit := range digits {
		if digit < '0' || digit > '9' {
			return false
		}
	}
	return true
}

func (account *AccountService) UpdateStatusSlug(ctx context.Context, userID domain.ID, slug string) (*domain.User, error) {
	if err := domain.ValidateSlug(slug); err != nil {
		return nil, err
	}
	user, err := account.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	user.StatusSlug = slug
	if err := account.users.Update(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (account *AccountService) SendTestAlert(ctx context.Context, userID domain.ID, channel domain.Channel) error {
	if channel != domain.ChannelEmail && channel != domain.ChannelTelegram {
		return apperrors.ValidationError{Field: "channel", Message: "is invalid"}
	}
	notifier := account.notifiers[channel]
	if notifier == nil {
		return apperrors.ErrProviderDisabled
	}
	user, err := account.users.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if !notifier.Enabled(*user) {
		return apperrors.ErrProviderDisabled
	}
	return notifier.Notify(ctx, *user, Alert{
		Monitor: domain.Monitor{Name: "Vigil test alert", Target: "Vigil"},
		Status:  domain.StatusDown,
		Detail:  "This is a test alert from Vigil.",
	})
}
