package service

import (
	"context"
	"errors"
	"testing"

	"github.com/endale-dev/vigil/internal/domain"
	apperrors "github.com/endale-dev/vigil/internal/errors"
)

func TestNewAccountServiceRequiresUsers(t *testing.T) {
	if account, err := NewAccountService(nil, false); err == nil || account != nil {
		t.Fatalf("NewAccountService() = (%v, %v), want dependency error", account, err)
	}
}

func TestAccountServiceUpdatesNotifications(t *testing.T) {
	users := &memoryUsers{stored: map[domain.ID]*domain.User{
		"user-1": {ID: "user-1", Notify: domain.NotifySettings{EmailAlerts: false}},
	}}
	account, err := NewAccountService(users, true)
	if err != nil {
		t.Fatal(err)
	}
	want := domain.NotifySettings{EmailAlerts: true, TelegramChatID: "-123456"}
	if err := account.UpdateNotifications(context.Background(), "user-1", want); err != nil {
		t.Fatalf("UpdateNotifications() error = %v", err)
	}
	if got := users.stored["user-1"].Notify; got != want {
		t.Fatalf("stored notifications = %+v, want %+v", got, want)
	}
}

func TestAccountServiceValidatesAndGatesTelegramChatID(t *testing.T) {
	for _, chatID := range []string{"1234", "123456789012345678901", "+12345", "12 345", "--12345"} {
		users := &memoryUsers{stored: map[domain.ID]*domain.User{"user-1": {ID: "user-1"}}}
		account, err := NewAccountService(users, true)
		if err != nil {
			t.Fatal(err)
		}
		if err := account.UpdateNotifications(context.Background(), "user-1", domain.NotifySettings{TelegramChatID: chatID}); err == nil {
			t.Errorf("UpdateNotifications(chatID=%q) succeeded", chatID)
		}
		if users.stored["user-1"].Notify.TelegramChatID != "" {
			t.Errorf("invalid chat ID %q was saved", chatID)
		}
	}
	users := &memoryUsers{stored: map[domain.ID]*domain.User{"user-1": {ID: "user-1"}}}
	account, err := NewAccountService(users, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := account.UpdateNotifications(context.Background(), "user-1", domain.NotifySettings{TelegramChatID: "12345"}); err != apperrors.ErrProviderDisabled {
		t.Fatalf("UpdateNotifications() error = %v, want ErrProviderDisabled", err)
	}
	if err := account.UpdateNotifications(context.Background(), "user-1", domain.NotifySettings{EmailAlerts: true}); err != nil {
		t.Fatalf("email-only update with Telegram disabled: %v", err)
	}
}

func TestAccountServiceUpdatesAndValidatesStatusSlug(t *testing.T) {
	users := &memoryUsers{stored: map[domain.ID]*domain.User{"user-1": {ID: "user-1", StatusSlug: "old-slug"}}}
	account, err := NewAccountService(users, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := account.UpdateStatusSlug(context.Background(), "user-1", "new-status"); err != nil {
		t.Fatalf("UpdateStatusSlug() error = %v", err)
	}
	if users.stored["user-1"].StatusSlug != "new-status" {
		t.Fatalf("status slug = %q, want new-status", users.stored["user-1"].StatusSlug)
	}
	if err := account.UpdateStatusSlug(context.Background(), "user-1", "Bad Slug"); err == nil {
		t.Fatal("UpdateStatusSlug() accepted invalid slug")
	}
	if users.stored["user-1"].StatusSlug != "new-status" {
		t.Fatal("invalid status slug changed the account")
	}
}

func TestAccountServicePropagatesRepositoryFailures(t *testing.T) {
	want := errors.New("repository unavailable")
	users := &memoryUsers{stored: map[domain.ID]*domain.User{"user-1": {ID: "user-1"}}, updateErr: want}
	account, err := NewAccountService(users, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := account.UpdateStatusSlug(context.Background(), "user-1", "new-status"); !errors.Is(err, want) {
		t.Fatalf("UpdateStatusSlug() error = %v, want %v", err, want)
	}
	if err := account.UpdateNotifications(context.Background(), "user-1", domain.NotifySettings{EmailAlerts: true}); !errors.Is(err, want) {
		t.Fatalf("UpdateNotifications() error = %v, want %v", err, want)
	}
}
