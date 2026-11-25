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
	updated, err := account.UpdateNotifications(context.Background(), "user-1", want)
	if err != nil {
		t.Fatalf("UpdateNotifications() error = %v", err)
	}
	if got := users.stored["user-1"].Notify; got != want {
		t.Fatalf("stored notifications = %+v, want %+v", got, want)
	}
	if updated.ID != "user-1" || updated.Notify != want {
		t.Fatalf("updated user = %+v", updated)
	}
}

func TestAccountServiceUpdatesAndValidatesProfileName(t *testing.T) {
	users := &memoryUsers{stored: map[domain.ID]*domain.User{"user-1": {ID: "user-1", Name: "Ada"}}}
	account, err := NewAccountService(users, false)
	if err != nil {
		t.Fatal(err)
	}
	updated, err := account.UpdateProfile(context.Background(), "user-1", "  Ada Lovelace  ")
	if err != nil {
		t.Fatalf("UpdateProfile() error = %v", err)
	}
	if updated.Name != "Ada Lovelace" || users.stored["user-1"].Name != "Ada Lovelace" {
		t.Fatalf("stored profile name = %q, returned %q", users.stored["user-1"].Name, updated.Name)
	}
	if _, err := account.UpdateProfile(context.Background(), "user-1", " "); err == nil {
		t.Fatal("UpdateProfile() accepted an empty name")
	}
	if users.stored["user-1"].Name != "Ada Lovelace" {
		t.Fatal("invalid profile update changed the saved name")
	}
}

func TestAccountServiceValidatesAndGatesTelegramChatID(t *testing.T) {
	for _, chatID := range []string{"1234", "123456789012345678901", "+12345", "12 345", "--12345"} {
		users := &memoryUsers{stored: map[domain.ID]*domain.User{"user-1": {ID: "user-1"}}}
		account, err := NewAccountService(users, true)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := account.UpdateNotifications(context.Background(), "user-1", domain.NotifySettings{TelegramChatID: chatID}); err == nil {
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
	if _, err := account.UpdateNotifications(context.Background(), "user-1", domain.NotifySettings{TelegramChatID: "12345"}); err != apperrors.ErrProviderDisabled {
		t.Fatalf("UpdateNotifications() error = %v, want ErrProviderDisabled", err)
	}
	if _, err := account.UpdateNotifications(context.Background(), "user-1", domain.NotifySettings{EmailAlerts: true}); err != nil {
		t.Fatalf("email-only update with Telegram disabled: %v", err)
	}
}

func TestAccountServiceUpdatesAndValidatesStatusSlug(t *testing.T) {
	users := &memoryUsers{stored: map[domain.ID]*domain.User{"user-1": {ID: "user-1", StatusSlug: "old-slug"}}}
	account, err := NewAccountService(users, false)
	if err != nil {
		t.Fatal(err)
	}
	updated, err := account.UpdateStatusSlug(context.Background(), "user-1", "new-status")
	if err != nil {
		t.Fatalf("UpdateStatusSlug() error = %v", err)
	}
	if users.stored["user-1"].StatusSlug != "new-status" || updated.StatusSlug != "new-status" {
		t.Fatalf("status slug = %q, want new-status", users.stored["user-1"].StatusSlug)
	}
	if _, err := account.UpdateStatusSlug(context.Background(), "user-1", "Bad Slug"); err == nil {
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
	if _, err := account.UpdateStatusSlug(context.Background(), "user-1", "new-status"); !errors.Is(err, want) {
		t.Fatalf("UpdateStatusSlug() error = %v, want %v", err, want)
	}
	if _, err := account.UpdateNotifications(context.Background(), "user-1", domain.NotifySettings{EmailAlerts: true}); !errors.Is(err, want) {
		t.Fatalf("UpdateNotifications() error = %v, want %v", err, want)
	}
}

func TestAccountServiceSendsTestAlertOnlyThroughRequestedChannel(t *testing.T) {
	users := &memoryUsers{stored: map[domain.ID]*domain.User{
		"user-1": {ID: "user-1", Notify: domain.NotifySettings{EmailAlerts: true, TelegramChatID: "12345"}},
	}}
	email := &fixedNotifier{channel: domain.ChannelEmail, enabled: true}
	telegram := &fixedNotifier{channel: domain.ChannelTelegram, enabled: true}
	account, err := NewAccountService(users, true, email, telegram)
	if err != nil {
		t.Fatal(err)
	}
	if err := account.SendTestAlert(context.Background(), "user-1", domain.ChannelEmail); err != nil {
		t.Fatalf("SendTestAlert() error = %v", err)
	}
	if email.calls != 1 || telegram.calls != 0 || len(email.alerts) != 1 || email.alerts[0].Status != domain.StatusDown {
		t.Fatalf("email/Telegram calls = %d/%d, email alerts = %+v", email.calls, telegram.calls, email.alerts)
	}
}

func TestAccountServiceRejectsUnknownUnavailableAndDisabledTestAlerts(t *testing.T) {
	users := &memoryUsers{stored: map[domain.ID]*domain.User{
		"user-1": {ID: "user-1", Notify: domain.NotifySettings{EmailAlerts: false}},
	}}
	disabledEmail := &fixedNotifier{channel: domain.ChannelEmail}
	account, err := NewAccountService(users, false, disabledEmail)
	if err != nil {
		t.Fatal(err)
	}
	if err := account.SendTestAlert(context.Background(), "user-1", domain.Channel("sms")); err == nil {
		t.Fatal("SendTestAlert() accepted an unknown channel")
	}
	if err := account.SendTestAlert(context.Background(), "user-1", domain.ChannelTelegram); err != apperrors.ErrProviderDisabled {
		t.Fatalf("unavailable notifier error = %v, want ErrProviderDisabled", err)
	}
	if err := account.SendTestAlert(context.Background(), "user-1", domain.ChannelEmail); err != apperrors.ErrProviderDisabled {
		t.Fatalf("disabled notifier error = %v, want ErrProviderDisabled", err)
	}
	if disabledEmail.calls != 0 {
		t.Fatalf("disabled notifier was called %d times", disabledEmail.calls)
	}
}

type fixedNotifier struct {
	channel domain.Channel
	enabled bool
	calls   int
	alerts  []Alert
	err     error
}

func (notifier *fixedNotifier) Name() domain.Channel { return notifier.channel }

func (notifier *fixedNotifier) Enabled(domain.User) bool { return notifier.enabled }

func (notifier *fixedNotifier) Notify(_ context.Context, _ domain.User, alert Alert) error {
	notifier.calls++
	notifier.alerts = append(notifier.alerts, alert)
	return notifier.err
}
