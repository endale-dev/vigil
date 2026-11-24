package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSessionCookieAttributes(t *testing.T) {
	expiresAt := time.Date(2026, 12, 1, 12, 0, 0, 0, time.UTC)
	response := httptest.NewRecorder()
	setSessionCookie(response, "session-token", expiresAt, true)
	cookies := response.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("got %d cookies, want 1", len(cookies))
	}
	cookie := cookies[0]
	if cookie.Name != sessionCookieName || cookie.Value != "session-token" {
		t.Fatalf("got session cookie %q=%q", cookie.Name, cookie.Value)
	}
	if cookie.Path != "/" || !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("session cookie has unsafe attributes: %#v", cookie)
	}
	if !cookie.Expires.Equal(expiresAt) {
		t.Fatalf("got expiration %v, want %v", cookie.Expires, expiresAt)
	}
}

func TestClearSessionCookie(t *testing.T) {
	response := httptest.NewRecorder()
	clearSessionCookie(response, false)
	cookies := response.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("got %d cookies, want 1", len(cookies))
	}
	cookie := cookies[0]
	if cookie.Name != sessionCookieName || cookie.MaxAge >= 0 || !cookie.Expires.Before(time.Now()) {
		t.Fatalf("session cookie was not expired: %#v", cookie)
	}
	if cookie.Path != "/" || !cookie.HttpOnly || cookie.Secure || cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("cleared session cookie has inconsistent attributes: %#v", cookie)
	}
}

func TestOAuthCookieAttributesAndLifetime(t *testing.T) {
	started := time.Now()
	response := httptest.NewRecorder()
	setOAuthCookies(response, "state-token", "pkce-verifier", true)
	cookies := response.Result().Cookies()
	if len(cookies) != 2 {
		t.Fatalf("got %d OAuth cookies, want 2", len(cookies))
	}
	wantValues := map[string]string{oauthStateCookie: "state-token", oauthVerifierCookie: "pkce-verifier"}
	for _, cookie := range cookies {
		if want, exists := wantValues[cookie.Name]; !exists || cookie.Value != want {
			t.Errorf("unexpected OAuth cookie %q=%q", cookie.Name, cookie.Value)
		}
		delete(wantValues, cookie.Name)
		if cookie.Path != "/" || !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode {
			t.Errorf("OAuth cookie has unsafe attributes: %#v", cookie)
		}
		if cookie.MaxAge != int(oauthCookieTTL.Seconds()) {
			t.Errorf("got MaxAge %d, want %d", cookie.MaxAge, int(oauthCookieTTL.Seconds()))
		}
		if cookie.Expires.Before(started.Add(oauthCookieTTL-time.Second)) || cookie.Expires.After(time.Now().Add(oauthCookieTTL+time.Second)) {
			t.Errorf("OAuth cookie expiration is outside its ten-minute lifetime: %v", cookie.Expires)
		}
	}
	if len(wantValues) != 0 {
		t.Errorf("missing OAuth cookies: %v", wantValues)
	}
}

func TestClearOAuthCookies(t *testing.T) {
	response := httptest.NewRecorder()
	clearOAuthCookies(response, true)
	cookies := response.Result().Cookies()
	if len(cookies) != 2 {
		t.Fatalf("got %d cookies, want 2", len(cookies))
	}
	for _, cookie := range cookies {
		if (cookie.Name != oauthStateCookie && cookie.Name != oauthVerifierCookie) || cookie.MaxAge >= 0 || !cookie.Expires.Before(time.Now()) {
			t.Errorf("OAuth cookie was not cleared: %#v", cookie)
		}
		if cookie.Path != "/" || !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode {
			t.Errorf("cleared OAuth cookie has inconsistent attributes: %#v", cookie)
		}
	}
}
