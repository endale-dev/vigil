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
	registerResult   service.AuthenticationResult
	registerError    error
	registerInput    registerRequest
	registerCalls    int
	loginResult      service.AuthenticationResult
	loginError       error
	loginInput       loginRequest
	loginCalls       int
	authResult       service.AuthenticatedSession
	authError        error
	authToken        string
	authCalls        int
	logoutError      error
	logoutSession    domain.ID
	logoutCalls      int
	googleStart      service.GoogleLoginStart
	googleStartErr   error
	googleStartCalls int
	googleResult     service.AuthenticationResult
	googleError      error
	googleCode       string
	googleVerifier   string
	googleCalls      int
	forgotError      error
	forgotEmail      string
	forgotCalls      int
	resetError       error
	resetToken       string
	resetPassword    string
	resetCalls       int
	changeError      error
	changeUserID     domain.ID
	changeSessionID  domain.ID
	currentPassword  string
	newPassword      string
	changeCalls      int
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

func (stub *authenticationStub) StartGoogleLogin() (service.GoogleLoginStart, error) {
	stub.googleStartCalls++
	return stub.googleStart, stub.googleStartErr
}

func (stub *authenticationStub) GoogleCallback(_ context.Context, code, verifier string) (service.AuthenticationResult, error) {
	stub.googleCalls++
	stub.googleCode = code
	stub.googleVerifier = verifier
	return stub.googleResult, stub.googleError
}

func (stub *authenticationStub) ForgotPassword(_ context.Context, email string) error {
	stub.forgotCalls++
	stub.forgotEmail = email
	return stub.forgotError
}

func (stub *authenticationStub) ResetPassword(_ context.Context, token, password string) error {
	stub.resetCalls++
	stub.resetToken = token
	stub.resetPassword = password
	return stub.resetError
}

func (stub *authenticationStub) ChangePassword(_ context.Context, userID, sessionID domain.ID, currentPassword, newPassword string) error {
	stub.changeCalls++
	stub.changeUserID = userID
	stub.changeSessionID = sessionID
	stub.currentPassword = currentPassword
	stub.newPassword = newPassword
	return stub.changeError
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

func TestGoogleStartRedirectsAndSetsOAuthCookies(t *testing.T) {
	serviceStub := &authenticationStub{googleStart: service.GoogleLoginStart{
		AuthorizationURL: "https://accounts.example/authorize", State: "oauth-state", CodeVerifier: "pkce-verifier",
	}}
	handler := newAuthenticationHandlerForTest(t, serviceStub, true)
	response := httptest.NewRecorder()
	handler.GoogleStart(response, httptest.NewRequest(http.MethodGet, "/api/auth/google/start", nil))

	if response.Code != http.StatusFound || response.Header().Get("Location") != serviceStub.googleStart.AuthorizationURL {
		t.Fatalf("Google start status/location = %d/%q", response.Code, response.Header().Get("Location"))
	}
	if serviceStub.googleStartCalls != 1 {
		t.Fatalf("Google start service calls = %d, want 1", serviceStub.googleStartCalls)
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 2 {
		t.Fatalf("got %d OAuth cookies, want 2", len(cookies))
	}
	want := map[string]string{oauthStateCookie: "oauth-state", oauthVerifierCookie: "pkce-verifier"}
	for _, cookie := range cookies {
		if cookie.Value != want[cookie.Name] || !cookie.Secure || !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode {
			t.Errorf("unexpected OAuth cookie: %#v", cookie)
		}
		delete(want, cookie.Name)
	}
	if len(want) != 0 {
		t.Errorf("missing OAuth cookies: %v", want)
	}
}

func TestGoogleStartReturnsNotFoundWhenProviderDisabled(t *testing.T) {
	serviceStub := &authenticationStub{googleStartErr: apperrors.ErrProviderDisabled}
	handler := newAuthenticationHandlerForTest(t, serviceStub, false)
	response := httptest.NewRecorder()
	handler.GoogleStart(response, httptest.NewRequest(http.MethodGet, "/api/auth/google/start", nil))
	if response.Code != http.StatusNotFound || len(response.Result().Cookies()) != 0 {
		t.Fatalf("Google start returned status %d and cookies %v", response.Code, response.Result().Cookies())
	}
}

func TestGoogleCallbackValidatesStateAndSetsSession(t *testing.T) {
	expiresAt := time.Date(2026, 12, 3, 12, 0, 0, 0, time.UTC)
	serviceStub := &authenticationStub{googleResult: service.AuthenticationResult{
		User:         domain.User{ID: "google-user", Name: "Ada", Email: "ada@example.com", GoogleSub: "private-sub"},
		SessionToken: "session-token", SessionExpiresAt: expiresAt,
	}}
	handler := newAuthenticationHandlerForTest(t, serviceStub, true)
	request := httptest.NewRequest(http.MethodGet, "/api/auth/google/callback?code=google-code&state=oauth-state", nil)
	request.AddCookie(&http.Cookie{Name: oauthStateCookie, Value: "oauth-state"})
	request.AddCookie(&http.Cookie{Name: oauthVerifierCookie, Value: "pkce-verifier"})
	response := httptest.NewRecorder()
	handler.GoogleCallback(response, request)

	if response.Code != http.StatusFound || response.Header().Get("Location") != "/" {
		t.Fatalf("Google callback status/location = %d/%q", response.Code, response.Header().Get("Location"))
	}
	if serviceStub.googleCalls != 1 || serviceStub.googleCode != "google-code" || serviceStub.googleVerifier != "pkce-verifier" {
		t.Fatalf("Google callback service args/calls = %q/%q/%d", serviceStub.googleCode, serviceStub.googleVerifier, serviceStub.googleCalls)
	}
	var sessionCookie *http.Cookie
	cleared := make(map[string]bool)
	for _, cookie := range response.Result().Cookies() {
		switch cookie.Name {
		case sessionCookieName:
			sessionCookie = cookie
		case oauthStateCookie, oauthVerifierCookie:
			cleared[cookie.Name] = cookie.MaxAge < 0
		}
	}
	if sessionCookie == nil || sessionCookie.Value != "session-token" || !sessionCookie.Expires.Equal(expiresAt) || !sessionCookie.Secure {
		t.Fatalf("unexpected Google session cookie: %#v", sessionCookie)
	}
	if !cleared[oauthStateCookie] || !cleared[oauthVerifierCookie] {
		t.Fatalf("OAuth cookies were not both cleared: %v", cleared)
	}
}

func TestGoogleCallbackFailureClearsCookiesWithoutAuthenticating(t *testing.T) {
	serviceStub := &authenticationStub{}
	handler := newAuthenticationHandlerForTest(t, serviceStub, true)
	request := httptest.NewRequest(http.MethodGet, "/api/auth/google/callback?code=google-code&state=attacker-state", nil)
	request.AddCookie(&http.Cookie{Name: oauthStateCookie, Value: "stored-state"})
	request.AddCookie(&http.Cookie{Name: oauthVerifierCookie, Value: "pkce-verifier"})
	response := httptest.NewRecorder()
	handler.GoogleCallback(response, request)

	if response.Code != http.StatusFound || response.Header().Get("Location") != "/login?error=google" {
		t.Fatalf("Google callback failure status/location = %d/%q", response.Code, response.Header().Get("Location"))
	}
	if serviceStub.googleCalls != 0 {
		t.Fatalf("Google callback service was called %d times on invalid state", serviceStub.googleCalls)
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 2 {
		t.Fatalf("got %d cookies, want two OAuth clear cookies", len(cookies))
	}
	for _, cookie := range cookies {
		if cookie.MaxAge >= 0 || !cookie.Secure {
			t.Errorf("OAuth cookie was not securely cleared: %#v", cookie)
		}
	}
}

func TestGoogleCallbackClearsCookiesWhenProviderFails(t *testing.T) {
	serviceStub := &authenticationStub{googleError: apperrors.ErrInvalidCredentials}
	handler := newAuthenticationHandlerForTest(t, serviceStub, false)
	request := httptest.NewRequest(http.MethodGet, "/api/auth/google/callback?code=google-code&state=oauth-state", nil)
	request.AddCookie(&http.Cookie{Name: oauthStateCookie, Value: "oauth-state"})
	request.AddCookie(&http.Cookie{Name: oauthVerifierCookie, Value: "pkce-verifier"})
	response := httptest.NewRecorder()
	handler.GoogleCallback(response, request)
	if response.Code != http.StatusFound || response.Header().Get("Location") != "/login?error=google" || serviceStub.googleCalls != 1 {
		t.Fatalf("Google provider failure status/location/calls = %d/%q/%d", response.Code, response.Header().Get("Location"), serviceStub.googleCalls)
	}
	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == sessionCookieName {
			t.Fatal("provider failure set a session cookie")
		}
		if cookie.MaxAge >= 0 {
			t.Errorf("OAuth cookie was not cleared: %#v", cookie)
		}
	}
}

func TestValidOAuthState(t *testing.T) {
	if !validOAuthState("same-state", "same-state") {
		t.Fatal("matching OAuth state was rejected")
	}
	for _, pair := range [][2]string{{"", ""}, {"short", "different-length"}, {"other-state", "same-state"}} {
		if validOAuthState(pair[0], pair[1]) {
			t.Errorf("accepted mismatched OAuth state %q and %q", pair[0], pair[1])
		}
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
