package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/endale-dev/vigil/internal/domain"
	apperrors "github.com/endale-dev/vigil/internal/errors"
	"github.com/endale-dev/vigil/internal/service"
)

func TestForgotPasswordHandlerReturnsNeutralResponse(t *testing.T) {
	serviceStub := &authenticationStub{}
	handler := newAuthenticationHandlerForTest(t, serviceStub, false)
	request := httptest.NewRequest(http.MethodPost, "/api/auth/password/forgot", strings.NewReader(`{"email":"person@example.com"}`))
	response := httptest.NewRecorder()
	handler.ForgotPassword(response, request)
	if response.Code != http.StatusAccepted || response.Body.String() != "{}\n" || serviceStub.forgotCalls != 1 || serviceStub.forgotEmail != "person@example.com" {
		t.Fatalf("forgot-password response/call = %d/%q/%d/%q", response.Code, response.Body.String(), serviceStub.forgotCalls, serviceStub.forgotEmail)
	}
}

func TestResetPasswordHandler(t *testing.T) {
	serviceStub := &authenticationStub{}
	handler := newAuthenticationHandlerForTest(t, serviceStub, false)
	request := httptest.NewRequest(http.MethodPost, "/api/auth/password/reset", strings.NewReader(`{"token":"reset-token","newPassword":"replacement-password"}`))
	response := httptest.NewRecorder()
	handler.ResetPassword(response, request)
	if response.Code != http.StatusNoContent || response.Body.Len() != 0 {
		t.Fatalf("reset-password status/body = %d/%q", response.Code, response.Body.String())
	}
	if serviceStub.resetCalls != 1 || serviceStub.resetToken != "reset-token" || serviceStub.resetPassword != "replacement-password" {
		t.Fatalf("reset-password call = %q/%q/%d", serviceStub.resetToken, serviceStub.resetPassword, serviceStub.resetCalls)
	}
}

func TestResetPasswordHandlerMapsInvalidToken(t *testing.T) {
	serviceStub := &authenticationStub{resetError: apperrors.ErrInvalidToken}
	handler := newAuthenticationHandlerForTest(t, serviceStub, false)
	request := httptest.NewRequest(http.MethodPost, "/api/auth/password/reset", strings.NewReader(`{"token":"expired","newPassword":"replacement-password"}`))
	response := httptest.NewRecorder()
	handler.ResetPassword(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("got status %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestChangePasswordHandlerAuthenticatesAndKeepsCurrentSession(t *testing.T) {
	serviceStub := &authenticationStub{authResult: service.AuthenticatedSession{
		User: domain.User{ID: "user-1"}, Session: domain.Session{ID: "session-1"},
	}}
	handler := newAuthenticationHandlerForTest(t, serviceStub, false)
	request := httptest.NewRequest(http.MethodPost, "/api/auth/password/change", strings.NewReader(`{"currentPassword":"old-password","newPassword":"new-password"}`))
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "session-token"})
	response := httptest.NewRecorder()
	handler.ChangePassword(response, request)

	if response.Code != http.StatusNoContent || response.Body.Len() != 0 {
		t.Fatalf("change-password status/body = %d/%q", response.Code, response.Body.String())
	}
	if serviceStub.authCalls != 1 || serviceStub.authToken != "session-token" || serviceStub.changeCalls != 1 ||
		serviceStub.changeUserID != "user-1" || serviceStub.changeSessionID != "session-1" ||
		serviceStub.currentPassword != "old-password" || serviceStub.newPassword != "new-password" {
		t.Fatalf("change-password auth/call state = %+v", serviceStub)
	}
}

func TestChangePasswordHandlerRequiresSession(t *testing.T) {
	serviceStub := &authenticationStub{}
	handler := newAuthenticationHandlerForTest(t, serviceStub, false)
	request := httptest.NewRequest(http.MethodPost, "/api/auth/password/change", strings.NewReader(`{"currentPassword":"old","newPassword":"new"}`))
	response := httptest.NewRecorder()
	handler.ChangePassword(response, request)
	if response.Code != http.StatusUnauthorized || serviceStub.authCalls != 0 || serviceStub.changeCalls != 0 {
		t.Fatalf("change-password status/auth/change calls = %d/%d/%d", response.Code, serviceStub.authCalls, serviceStub.changeCalls)
	}
}

func TestChangePasswordHandlerReportsIncorrectCurrentPasswordAsFieldError(t *testing.T) {
	serviceStub := &authenticationStub{
		authResult: service.AuthenticatedSession{
			User: domain.User{ID: "user-1"}, Session: domain.Session{ID: "session-1"},
		},
		changeError: apperrors.ErrInvalidCredentials,
	}
	handler := newAuthenticationHandlerForTest(t, serviceStub, false)
	request := httptest.NewRequest(http.MethodPost, "/api/auth/password/change", strings.NewReader(`{"currentPassword":"wrong","newPassword":"replacement-password"}`))
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "session-token"})
	response := httptest.NewRecorder()
	handler.ChangePassword(response, request)

	if response.Code != http.StatusUnprocessableEntity || !strings.Contains(response.Body.String(), `"field":"currentPassword"`) {
		t.Fatalf("change-password status/body = %d/%q", response.Code, response.Body.String())
	}
}
