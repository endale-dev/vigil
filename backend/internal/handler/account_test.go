package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/endale-dev/vigil/internal/domain"
	apperrors "github.com/endale-dev/vigil/internal/errors"
	"github.com/endale-dev/vigil/internal/service"
)

type accountServiceStub struct {
	user              *domain.User
	updateNotifyError error
	updateNotify      domain.NotifySettings
	updateNotifyID    domain.ID
	updateNotifyCalls int
	updateSlugError   error
	updateSlug        string
	updateSlugID      domain.ID
	updateSlugCalls   int
	testAlertError    error
	testAlertChannel  domain.Channel
	testAlertID       domain.ID
	testAlertCalls    int
}

func (stub *accountServiceStub) UpdateNotifications(_ context.Context, userID domain.ID, settings domain.NotifySettings) (*domain.User, error) {
	stub.updateNotifyCalls++
	stub.updateNotifyID = userID
	stub.updateNotify = settings
	return stub.user, stub.updateNotifyError
}

func (stub *accountServiceStub) UpdateStatusSlug(_ context.Context, userID domain.ID, slug string) (*domain.User, error) {
	stub.updateSlugCalls++
	stub.updateSlugID = userID
	stub.updateSlug = slug
	return stub.user, stub.updateSlugError
}

func (stub *accountServiceStub) SendTestAlert(_ context.Context, userID domain.ID, channel domain.Channel) error {
	stub.testAlertCalls++
	stub.testAlertID = userID
	stub.testAlertChannel = channel
	return stub.testAlertError
}

func TestUpdateNotificationsHandlerReturnsUpdatedUser(t *testing.T) {
	wantNotify := domain.NotifySettings{EmailAlerts: false, TelegramChatID: "-123456"}
	accountStub := &accountServiceStub{user: &domain.User{
		ID: "user-1", Name: "Ada", Email: "ada@example.com", PasswordHash: "private-hash",
		StatusSlug: "status-1", Notify: wantNotify,
	}}
	handler := newAccountHandlerForTest(t, accountStub, &authenticationStub{authResult: service.AuthenticatedSession{
		User: domain.User{ID: "user-1"}, HasPassword: true,
	}})
	request := httptest.NewRequest(http.MethodPut, "/api/me/notifications", strings.NewReader(`{"emailAlerts":false,"telegramChatId":"-123456"}`))
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "session-token"})
	response := httptest.NewRecorder()
	handler.UpdateNotifications(response, request)

	if response.Code != http.StatusOK || accountStub.updateNotifyCalls != 1 || accountStub.updateNotifyID != "user-1" || accountStub.updateNotify != wantNotify {
		t.Fatalf("notification update status/call = %d/%+v", response.Code, accountStub)
	}
	var user UserDTO
	if err := json.Unmarshal(response.Body.Bytes(), &user); err != nil {
		t.Fatal(err)
	}
	if user.ID != "user-1" || user.HasPassword != true || user.Notify != (NotifyDTO{EmailAlerts: false, TelegramChatID: "-123456"}) {
		t.Fatalf("unexpected updated user DTO: %+v", user)
	}
	if strings.Contains(response.Body.String(), "private-hash") {
		t.Fatal("notification response exposed the password hash")
	}
}

func TestUpdateNotificationsHandlerMapsDisabledTelegram(t *testing.T) {
	accountStub := &accountServiceStub{updateNotifyError: apperrors.ErrProviderDisabled}
	handler := newAccountHandlerForTest(t, accountStub, &authenticationStub{authResult: service.AuthenticatedSession{User: domain.User{ID: "user-1"}}})
	request := httptest.NewRequest(http.MethodPut, "/api/me/notifications", strings.NewReader(`{"emailAlerts":true,"telegramChatId":"12345"}`))
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "session-token"})
	response := httptest.NewRecorder()
	handler.UpdateNotifications(response, request)
	if response.Code != http.StatusNotFound || accountStub.updateNotifyCalls != 1 {
		t.Fatalf("notification update status/calls = %d/%d", response.Code, accountStub.updateNotifyCalls)
	}
}

func TestUpdateStatusSlugHandlerReturnsUpdatedUser(t *testing.T) {
	accountStub := &accountServiceStub{user: &domain.User{ID: "user-1", StatusSlug: "new-status"}}
	handler := newAccountHandlerForTest(t, accountStub, &authenticationStub{authResult: service.AuthenticatedSession{User: domain.User{ID: "user-1"}}})
	request := httptest.NewRequest(http.MethodPut, "/api/me/status-slug", strings.NewReader(`{"statusSlug":"new-status"}`))
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "session-token"})
	response := httptest.NewRecorder()
	handler.UpdateStatusSlug(response, request)
	if response.Code != http.StatusOK || accountStub.updateSlugCalls != 1 || accountStub.updateSlugID != "user-1" || accountStub.updateSlug != "new-status" {
		t.Fatalf("status-slug update = %d/%+v", response.Code, accountStub)
	}
	var user UserDTO
	if err := json.Unmarshal(response.Body.Bytes(), &user); err != nil {
		t.Fatal(err)
	}
	if user.StatusSlug != "new-status" {
		t.Fatalf("got status slug %q", user.StatusSlug)
	}
}

func TestSendTestAlertHandlerUsesRequestedChannel(t *testing.T) {
	accountStub := &accountServiceStub{}
	handler := newAccountHandlerForTest(t, accountStub, &authenticationStub{authResult: service.AuthenticatedSession{User: domain.User{ID: "user-1"}}})
	request := httptest.NewRequest(http.MethodPost, "/api/me/notifications/test/email", nil)
	request.SetPathValue("channel", "email")
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "session-token"})
	response := httptest.NewRecorder()
	handler.SendTestAlert(response, request)
	if response.Code != http.StatusNoContent || accountStub.testAlertCalls != 1 || accountStub.testAlertID != "user-1" || accountStub.testAlertChannel != domain.ChannelEmail {
		t.Fatalf("test-alert result = %d/%+v", response.Code, accountStub)
	}
}

func TestAccountHandlersRequireAuthentication(t *testing.T) {
	accountStub := &accountServiceStub{}
	authStub := &authenticationStub{authError: apperrors.ErrUnauthenticated}
	handler := newAccountHandlerForTest(t, accountStub, authStub)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/me/notifications/test/email", nil)
	request.SetPathValue("channel", "email")
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "bad-session"})
	handler.SendTestAlert(response, request)
	if response.Code != http.StatusUnauthorized || accountStub.testAlertCalls != 0 || authStub.authCalls != 1 {
		t.Fatalf("test-alert auth status/calls = %d/%d/%d", response.Code, authStub.authCalls, accountStub.testAlertCalls)
	}
}

func TestNewAccountHandlerRequiresDependencies(t *testing.T) {
	if handler, err := NewAccountHandler(nil, &accountServiceStub{}, CapabilitiesDTO{}, slog.New(slog.NewTextHandler(io.Discard, nil))); err == nil || handler != nil {
		t.Fatalf("NewAccountHandler() = (%v, %v), want dependency error", handler, err)
	}
	if handler, err := NewAccountHandler(&authenticationStub{}, nil, CapabilitiesDTO{}, slog.New(slog.NewTextHandler(io.Discard, nil))); err == nil || handler != nil {
		t.Fatalf("NewAccountHandler() = (%v, %v), want dependency error", handler, err)
	}
	if handler, err := NewAccountHandler(&authenticationStub{}, &accountServiceStub{}, CapabilitiesDTO{}, nil); err == nil || handler != nil {
		t.Fatalf("NewAccountHandler() = (%v, %v), want dependency error", handler, err)
	}
}

func TestAccountServiceErrorsAreMapped(t *testing.T) {
	accountStub := &accountServiceStub{updateSlugError: errors.New("database secret")}
	handler := newAccountHandlerForTest(t, accountStub, &authenticationStub{authResult: service.AuthenticatedSession{User: domain.User{ID: "user-1"}}})
	request := httptest.NewRequest(http.MethodPut, "/api/me/status-slug", strings.NewReader(`{"statusSlug":"valid-slug"}`))
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "session-token"})
	response := httptest.NewRecorder()
	handler.UpdateStatusSlug(response, request)
	if response.Code != http.StatusInternalServerError || strings.Contains(response.Body.String(), "database secret") {
		t.Fatalf("status update leaked an internal error: %d %s", response.Code, response.Body.String())
	}
}

func newAccountHandlerForTest(t *testing.T, accountStub accountService, authStub sessionAuthenticator) *AccountHandler {
	t.Helper()
	handler, err := NewAccountHandler(authStub, accountStub, CapabilitiesDTO{Google: true, Telegram: true}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	return handler
}
