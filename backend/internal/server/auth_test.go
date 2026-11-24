package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/endale-dev/vigil/internal/domain"
	apperrors "github.com/endale-dev/vigil/internal/errors"
	"github.com/endale-dev/vigil/internal/service"
)

type sessionAuthenticatorStub struct {
	session service.AuthenticatedSession
	err     error
	token   string
	calls   int
}

func (stub *sessionAuthenticatorStub) Authenticate(_ context.Context, token string) (service.AuthenticatedSession, error) {
	stub.calls++
	stub.token = token
	return stub.session, stub.err
}

func TestRequireAuthAddsAuthenticatedUserToContext(t *testing.T) {
	authentication := &sessionAuthenticatorStub{session: service.AuthenticatedSession{User: domain.User{ID: "user-1", Email: "ada@example.test"}}}
	next := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		user, ok := UserFrom(request.Context())
		if !ok || user.ID != "user-1" || user.Email != "ada@example.test" {
			t.Errorf("UserFrom() = (%+v, %v), want authenticated user", user, ok)
		}
		writer.WriteHeader(http.StatusNoContent)
	})
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "raw-session-token"})
	response := httptest.NewRecorder()
	RequireAuth(authentication)(next).ServeHTTP(response, request)

	if response.Code != http.StatusNoContent || authentication.calls != 1 || authentication.token != "raw-session-token" {
		t.Fatalf("response/authentication = %d/%d/%q", response.Code, authentication.calls, authentication.token)
	}
	if user, ok := UserFrom(context.Background()); ok || user.ID != "" {
		t.Fatalf("UserFrom(empty context) = (%+v, %v), want no user", user, ok)
	}
}

func TestRequireAuthRejectsMissingOrInvalidSessions(t *testing.T) {
	for _, test := range []struct {
		name   string
		err    error
		cookie bool
	}{
		{name: "missing cookie"},
		{name: "invalid session", err: apperrors.ErrUnauthenticated, cookie: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			authentication := &sessionAuthenticatorStub{err: test.err}
			nextCalls := 0
			next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { nextCalls++ })
			request := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if test.cookie {
				request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "bad-token"})
			}
			response := httptest.NewRecorder()
			RequireAuth(authentication)(next).ServeHTTP(response, request)
			if response.Code != http.StatusUnauthorized || nextCalls != 0 {
				t.Fatalf("response/next calls = %d/%d, want 401/0", response.Code, nextCalls)
			}
			if !test.cookie && authentication.calls != 0 {
				t.Fatalf("authenticator called %d times without a cookie", authentication.calls)
			}
		})
	}
}

func TestRequireAuthHidesUnexpectedAuthenticationErrors(t *testing.T) {
	authentication := &sessionAuthenticatorStub{err: errors.New("database credentials leaked")}
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "token"})
	response := httptest.NewRecorder()
	RequireAuth(authentication)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("protected handler was called after authentication failure")
	})).ServeHTTP(response, request)
	if response.Code != http.StatusInternalServerError || response.Body.String() != "{\"error\":\"internal server error\"}\n" {
		t.Fatalf("unexpected auth error response = %d, %q", response.Code, response.Body.String())
	}
}

func TestRequireAuthRejectsMissingAuthenticator(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	response := httptest.NewRecorder()
	RequireAuth(nil)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("protected handler was called without an authenticator")
	})).ServeHTTP(response, request)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("missing authenticator status = %d, want 500", response.Code)
	}
}
