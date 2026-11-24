package handler

import (
	"net/http"
	"time"
)

const (
	sessionCookieName   = "vigil_session"
	oauthStateCookie    = "oauth_state"
	oauthVerifierCookie = "oauth_verifier"
	oauthCookieTTL      = 10 * time.Minute
)

func setSessionCookie(writer http.ResponseWriter, token string, expiresAt time.Time, secure bool) {
	http.SetCookie(writer, &http.Cookie{
		Name: sessionCookieName, Value: token, Path: "/", Expires: expiresAt,
		HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode,
	})
}

func clearSessionCookie(writer http.ResponseWriter, secure bool) {
	http.SetCookie(writer, &http.Cookie{
		Name: sessionCookieName, Path: "/", Expires: time.Unix(1, 0), MaxAge: -1,
		HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode,
	})
}

func setOAuthCookies(writer http.ResponseWriter, state, verifier string, secure bool) {
	expiresAt := time.Now().Add(oauthCookieTTL)
	for _, cookie := range []*http.Cookie{
		{Name: oauthStateCookie, Value: state},
		{Name: oauthVerifierCookie, Value: verifier},
	} {
		cookie.Path = "/"
		cookie.Expires = expiresAt
		cookie.MaxAge = int(oauthCookieTTL.Seconds())
		cookie.HttpOnly = true
		cookie.Secure = secure
		cookie.SameSite = http.SameSiteLaxMode
		http.SetCookie(writer, cookie)
	}
}

func clearOAuthCookies(writer http.ResponseWriter, secure bool) {
	for _, name := range []string{oauthStateCookie, oauthVerifierCookie} {
		http.SetCookie(writer, &http.Cookie{
			Name: name, Path: "/", Expires: time.Unix(1, 0), MaxAge: -1,
			HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode,
		})
	}
}
