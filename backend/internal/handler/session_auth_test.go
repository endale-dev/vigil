package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/endale-dev/vigil/internal/authcontext"
	"github.com/endale-dev/vigil/internal/domain"
	"github.com/endale-dev/vigil/internal/service"
)

func TestAuthenticatedSessionReusesMiddlewareSession(t *testing.T) {
	want := service.AuthenticatedSession{User: domain.User{ID: "user-1"}, HasPassword: true}
	authentication := &authenticationStub{}
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request = request.WithContext(authcontext.WithSession(request.Context(), want))
	session, err := authenticatedSession(request, authentication)
	if err != nil || session.User.ID != want.User.ID || !session.HasPassword || authentication.authCalls != 0 {
		t.Fatalf("authenticatedSession() = (%+v, %v), auth calls %d", session, err, authentication.authCalls)
	}
}

func TestAuthenticatedSessionFallsBackToCookie(t *testing.T) {
	want := service.AuthenticatedSession{User: domain.User{ID: "user-1"}}
	authentication := &authenticationStub{authResult: want}
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "session-token"})
	session, err := authenticatedSession(request, authentication)
	if err != nil || session.User.ID != want.User.ID || authentication.authCalls != 1 || authentication.authToken != "session-token" {
		t.Fatalf("authenticatedSession() = (%+v, %v), auth calls/token %d/%q", session, err, authentication.authCalls, authentication.authToken)
	}
	if _, err := authenticatedSession(httptest.NewRequest(http.MethodGet, "/protected", nil), authentication); err == nil {
		t.Fatal("authenticatedSession() without a context session or cookie returned no error")
	}
}
