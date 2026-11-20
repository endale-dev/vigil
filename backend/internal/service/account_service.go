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
}

func NewAccountService(users domain.UserRepository, telegramEnabled bool) (*AccountService, error) {
	if users == nil {
		return nil, errors.New("account service user repository is required")
	}
	return &AccountService{users: users, telegramEnabled: telegramEnabled}, nil
}

func (account *AccountService) UpdateNotifications(ctx context.Context, userID domain.ID, notify domain.NotifySettings) error {
	if notify.TelegramChatID != "" {
		if !account.telegramEnabled {
			return apperrors.ErrProviderDisabled
		}
		if !validTelegramChatID(notify.TelegramChatID) {
			return apperrors.ValidationError{Field: "telegramChatId", Message: "must be empty or a 5 to 20 digit chat ID"}
		}
	}
	user, err := account.users.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	user.Notify = notify
	return account.users.Update(ctx, user)
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

func (account *AccountService) UpdateStatusSlug(ctx context.Context, userID domain.ID, slug string) error {
	if err := domain.ValidateSlug(slug); err != nil {
		return err
	}
	user, err := account.users.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	user.StatusSlug = slug
	return account.users.Update(ctx, user)
}
