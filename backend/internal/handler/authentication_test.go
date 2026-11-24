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
	"time"

	"github.com/endale-dev/vigil/internal/domain"
	apperrors "github.com/endale-dev/vigil/internal/errors"
	"github.com/endale-dev/vigil/internal/service"
)

type authenticationStub struct {
	registerResult service.AuthenticationResult
	registerError  error
	registerInput  registerRequest
	registerCalls  int
	loginResult    service.AuthenticationResult
	loginError     error
	loginInput     loginRequest
	loginCalls     int
	authResult     service.AuthenticatedSession
	authError      error
	authToken      string
	authCalls      int
	logoutError    error
	logoutSession  domain.ID
	logoutCalls    int
}

func (stub *authenticationStub) Register(_ context.Context, name, email, password string) (service.AuthenticationResult, error) {
	stub.registerCalls++
	stub.registerInput = registerRequest{Name: name, Email: email, Password: password}
	return stub.registerResult, stub.registerError
}

func (stub *authenticationStub) Login(_ context.Context, email, password string) (service.AuthenticationResult, error) {
	stub.loginCalls++
	stub.loginInput = loginRequest{Email: email, Password: password}
	return stub.loginResult, stub.loginError
}

func (stub *authenticationStub) Authenticate(_ context.Context, token string) (service.AuthenticatedSession, error) {
	stub.authCalls++
	stub.authToken = token
	return stub.authResult, stub.authError
}

func (stub *authenticationStub) Logout(_ context.Context, sessionID domain.ID) error {
	stub.logoutCalls++
	stub.logoutSession = sessionID
	return stub.logoutError
}

func TestRegisterHandlerCreatesSessionCookieAndUserDTO(t *testing.T) {
	expiresAt := time.Date(2026, 12, 1, 12, 0, 0, 0, time.UTC)
	serviceStub := &authenticationStub{registerResult: service.AuthenticationResult{
		User:         domain.User{ID: "user-1", Name: "Ada", Email: "ada@example.com", PasswordHash: "private-hash", StatusSlug: "status-1"},
		SessionToken: "session-token", SessionExpiresAt: expiresAt, HasPassword: true,
	}}
	handler := newAuthenticationHandlerForTest(t, serviceStub, true)
	request := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(`{"name":"Ada","email":"ada@example.com","password":"correct horse"}`))
	response := httptest.NewRecorder()
	handler.Register(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("got status %d, want %d", response.Code, http.StatusCreated)
	}
	if serviceStub.registerCalls != 1 || serviceStub.registerInput != (registerRequest{Name: "Ada", Email: "ada@example.com", Password: "correct horse"}) {
		t.Fatalf("register call = %+v (%d calls)", serviceStub.registerInput, serviceStub.registerCalls)
	}
	cookie := sessionCookieFromResponse(t, response)
	if cookie.Value != "session-token" || !cookie.Expires.Equal(expiresAt) || !cookie.Secure {
		t.Fatalf("unexpected registration session cookie: %#v", cookie)
	}
	var user UserDTO
	if err := json.Unmarshal(response.Body.Bytes(), &user); err != nil {
		t.Fatal(err)
	}
	if user.ID != "user-1" || !user.HasPassword || !user.Capabilities.Google || !user.Capabilities.Telegram {
		t.Fatalf("unexpected registration user DTO: %+v", user)
	}
	if strings.Contains(response.Body.String(), "private-hash") || strings.Contains(response.Body.String(), "session-token") {
		t.Fatalf("registration response exposed credentials: %s", response.Body.String())
	}
}

func TestLoginHandlerCreatesSessionCookieAndUserDTO(t *testing.T) {
	expiresAt := time.Date(2026, 12, 2, 12, 0, 0, 0, time.UTC)
	serviceStub := &authenticationStub{loginResult: service.AuthenticationResult{
		User:         domain.User{ID: "user-2", Name: "Grace", Email: "grace@example.com"},
		SessionToken: "session-token", SessionExpiresAt: expiresAt, HasPassword: true,
	}}
	handler := newAuthenticationHandlerForTest(t, serviceStub, false)
	request := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"email":"grace@example.com","password":"correct horse"}`))
	response := httptest.NewRecorder()
	handler.Login(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d", response.Code, http.StatusOK)
	}
	if serviceStub.loginCalls != 1 || serviceStub.loginInput != (loginRequest{Email: "grace@example.com", Password: "correct horse"}) {
		t.Fatalf("login call = %+v (%d calls)", serviceStub.loginInput, serviceStub.loginCalls)
	}
	cookie := sessionCookieFromResponse(t, response)
	if cookie.Value != "session-token" || !cookie.Expires.Equal(expiresAt) || cookie.Secure {
		t.Fatalf("unexpected login session cookie: %#v", cookie)
	}
}

func TestLoginHandlerMapsErrorsWithoutSettingSession(t *testing.T) {
	serviceStub := &authenticationStub{loginError: apperrors.ErrInvalidCredentials}
	handler := newAuthenticationHandlerForTest(t, serviceStub, false)
	request := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"email":"grace@example.com","password":"wrong"}`))
	response := httptest.NewRecorder()
	handler.Login(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("got status %d, want %d", response.Code, http.StatusUnauthorized)
	}
	if len(response.Result().Cookies()) != 0 {
		t.Fatal("login error set a session cookie")
	}
	if serviceStub.loginCalls != 1 {
		t.Fatalf("service was called %d times, want once", serviceStub.loginCalls)
	}
}

func TestRegisterHandlerRejectsInvalidJSONBeforeCallingService(t *testing.T) {
	serviceStub := &authenticationStub{}
	handler := newAuthenticationHandlerForTest(t, serviceStub, false)
	request := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(`{"name":"Ada","email":"ada@example.com","password":"secret","unexpected":true}`))
	response := httptest.NewRecorder()
	handler.Register(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("got status %d, want %d", response.Code, http.StatusBadRequest)
	}
	if serviceStub.registerCalls != 0 || len(response.Result().Cookies()) != 0 {
		t.Fatal("invalid request reached the service or set a session cookie")
	}
}

func TestMeHandlerReturnsAuthenticatedUserDTO(t *testing.T) {
	serviceStub := &authenticationStub{authResult: service.AuthenticatedSession{
		User:    domain.User{ID: "user-3", Name: "Lin", Email: "lin@example.com", GoogleSub: "google-secret"},
		Session: domain.Session{ID: "session-3"}, HasPassword: true,
	}}
	handler := newAuthenticationHandlerForTest(t, serviceStub, true)
	request := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "session-token"})
	response := httptest.NewRecorder()
	handler.Me(response, request)

	if response.Code != http.StatusOK || serviceStub.authCalls != 1 || serviceStub.authToken != "session-token" {
		t.Fatalf("me status/calls/token = %d/%d/%q", response.Code, serviceStub.authCalls, serviceStub.authToken)
	}
	var user UserDTO
	if err := json.Unmarshal(response.Body.Bytes(), &user); err != nil {
		t.Fatal(err)
	}
	if user.ID != "user-3" || !user.HasPassword || !user.GoogleLinked || !user.Capabilities.Google || !user.Capabilities.Telegram {
		t.Fatalf("unexpected current-user DTO: %+v", user)
	}
	if strings.Contains(response.Body.String(), "google-secret") || strings.Contains(response.Body.String(), "session-token") {
		t.Fatalf("current-user response exposed credentials: %s", response.Body.String())
	}
}

func TestMeHandlerRequiresSessionCookie(t *testing.T) {
	serviceStub := &authenticationStub{}
	handler := newAuthenticationHandlerForTest(t, serviceStub, false)
	response := httptest.NewRecorder()
	handler.Me(response, httptest.NewRequest(http.MethodGet, "/api/me", nil))
	if response.Code != http.StatusUnauthorized || serviceStub.authCalls != 0 {
		t.Fatalf("me status/service calls = %d/%d, want 401/0", response.Code, serviceStub.authCalls)
	}
}

func TestLogoutHandlerClearsAbsentOrInvalidSession(t *testing.T) {
	for _, test := range []struct {
		name      string
		token     string
		authError error
		wantAuth  int
	}{
		{name: "absent session"},
		{name: "invalid session", token: "expired-token", authError: apperrors.ErrUnauthenticated, wantAuth: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			serviceStub := &authenticationStub{authError: test.authError}
			handler := newAuthenticationHandlerForTest(t, serviceStub, true)
			request := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
			if test.token != "" {
				request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: test.token})
			}
			response := httptest.NewRecorder()
			handler.Logout(response, request)

			if response.Code != http.StatusNoContent || serviceStub.authCalls != test.wantAuth || serviceStub.logoutCalls != 0 {
				t.Fatalf("logout status/auth/logout calls = %d/%d/%d", response.Code, serviceStub.authCalls, serviceStub.logoutCalls)
			}
			cookie := sessionCookieFromResponse(t, response)
			if cookie.MaxAge >= 0 || !cookie.Secure {
				t.Fatalf("session cookie was not securely cleared: %#v", cookie)
			}
		})
	}
}

func TestLogoutHandlerDeletesSessionAndClearsCookie(t *testing.T) {
	serviceStub := &authenticationStub{authResult: service.AuthenticatedSession{Session: domain.Session{ID: "session-4"}}}
	handler := newAuthenticationHandlerForTest(t, serviceStub, false)
	request := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "session-token"})
	response := httptest.NewRecorder()
	handler.Logout(response, request)

	if response.Code != http.StatusNoContent || serviceStub.authCalls != 1 || serviceStub.logoutCalls != 1 || serviceStub.logoutSession != "session-4" {
		t.Fatalf("logout status/auth/logout/session = %d/%d/%d/%q", response.Code, serviceStub.authCalls, serviceStub.logoutCalls, serviceStub.logoutSession)
	}
	cookie := sessionCookieFromResponse(t, response)
	if cookie.MaxAge >= 0 || cookie.Secure {
		t.Fatalf("session cookie was not cleared with development settings: %#v", cookie)
	}
}

func TestLogoutHandlerPreservesCookieWhenSessionDeletionFails(t *testing.T) {
	serviceStub := &authenticationStub{
		authResult:  service.AuthenticatedSession{Session: domain.Session{ID: "session-5"}},
		logoutError: errors.New("session store unavailable"),
	}
	handler := newAuthenticationHandlerForTest(t, serviceStub, false)
	request := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "session-token"})
	response := httptest.NewRecorder()
	handler.Logout(response, request)
	if response.Code != http.StatusInternalServerError || len(response.Result().Cookies()) != 0 {
		t.Fatalf("logout failure returned %d with cookies %v", response.Code, response.Result().Cookies())
	}
}

func newAuthenticationHandlerForTest(t *testing.T, serviceStub authenticationService, secure bool) *AuthenticationHandler {
	t.Helper()
	handler, err := NewAuthenticationHandler(serviceStub, secure, CapabilitiesDTO{Google: true, Telegram: true}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	return handler
}

func sessionCookieFromResponse(t *testing.T, response *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == sessionCookieName {
			return cookie
		}
	}
	t.Fatal("response has no session cookie")
	return nil
}
