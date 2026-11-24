package google

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestProviderAuthURLIncludesOAuthAndPKCEParameters(t *testing.T) {
	provider := NewProvider("client id", "secret", "https://vigil.example.test/api/auth/google/callback")
	parsed, err := url.Parse(provider.AuthURL("state-value", "challenge-value"))
	if err != nil {
		t.Fatalf("parse AuthURL(): %v", err)
	}
	query := parsed.Query()
	for key, want := range map[string]string{
		"client_id":             "client id",
		"redirect_uri":          "https://vigil.example.test/api/auth/google/callback",
		"response_type":         "code",
		"scope":                 "openid email profile",
		"state":                 "state-value",
		"code_challenge":        "challenge-value",
		"code_challenge_method": "S256",
	} {
		if got := query.Get(key); got != want {
			t.Errorf("AuthURL parameter %q = %q, want %q", key, got, want)
		}
	}
}

func TestProviderExchangesCodeAndFetchesVerifiedIdentity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/token":
			if request.Method != http.MethodPost || request.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
				t.Errorf("token request = %s %s, Content-Type %q", request.Method, request.URL.Path, request.Header.Get("Content-Type"))
			}
			if err := request.ParseForm(); err != nil {
				t.Errorf("parse token form: %v", err)
			}
			for key, want := range map[string]string{
				"code":          "authorization-code",
				"client_id":     "client-id",
				"client_secret": "client-secret",
				"redirect_uri":  "https://vigil.example.test/api/auth/google/callback",
				"grant_type":    "authorization_code",
				"code_verifier": "verifier-value",
			} {
				if got := request.Form.Get(key); got != want {
					t.Errorf("token form %q = %q, want %q", key, got, want)
				}
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"access-secret","token_type":"Bearer"}`))
		case "/userinfo":
			if request.Method != http.MethodGet || request.Header.Get("Authorization") != "Bearer access-secret" {
				t.Errorf("userinfo request = %s, Authorization %q", request.Method, request.Header.Get("Authorization"))
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"sub":"google-sub","email":"user@example.test","name":"Example User","email_verified":true}`))
		default:
			http.NotFound(w, request)
		}
	}))
	defer server.Close()

	provider := NewProvider("client-id", "client-secret", "https://vigil.example.test/api/auth/google/callback")
	provider.tokenEndpoint = server.URL + "/token"
	provider.userInfoEndpoint = server.URL + "/userinfo"
	provider.client.Timeout = time.Second
	identity, err := provider.Exchange(context.Background(), "authorization-code", "verifier-value")
	if err != nil {
		t.Fatalf("Exchange() error = %v", err)
	}
	if identity.Sub != "google-sub" || identity.Email != "user@example.test" || identity.Name != "Example User" || !identity.EmailVerified {
		t.Fatalf("Exchange() = %+v, want verified Google identity", identity)
	}
}

func TestProviderRejectsInvalidIdentityAndSanitizesErrors(t *testing.T) {
	for _, test := range []struct {
		name       string
		userInfo   string
		status     int
		wantErr    string
		wantSubstr string
	}{
		{name: "unverified", userInfo: `{"sub":"subject","email":"user@example.test","email_verified":false}`, status: http.StatusOK, wantSubstr: "not verified"},
		{name: "missing subject", userInfo: `{"email":"user@example.test","email_verified":true}`, status: http.StatusOK, wantSubstr: "not verified"},
		{name: "token endpoint failure", status: http.StatusBadGateway, wantErr: "google token exchange failed"},
		{name: "userinfo endpoint failure", userInfo: `{"error":"access-secret"}`, status: http.StatusBadGateway, wantErr: "google identity lookup failed"},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
				if request.URL.Path == "/token" {
					if test.name == "token endpoint failure" {
						w.WriteHeader(test.status)
						_, _ = w.Write([]byte("client-secret authorization-code"))
						return
					}
					_, _ = w.Write([]byte(`{"access_token":"access-secret"}`))
					return
				}
				w.WriteHeader(test.status)
				_, _ = w.Write([]byte(test.userInfo))
			}))
			defer server.Close()
			provider := NewProvider("client-id", "client-secret", "https://vigil.example.test/callback")
			provider.tokenEndpoint = server.URL + "/token"
			provider.userInfoEndpoint = server.URL + "/userinfo"
			identity, err := provider.Exchange(context.Background(), "authorization-code", "verifier")
			if err == nil || identity.Sub != "" {
				t.Fatalf("Exchange() = (%+v, %v), want failure", identity, err)
			}
			if test.wantErr != "" && err.Error() != test.wantErr {
				t.Fatalf("Exchange() error = %q, want %q", err, test.wantErr)
			}
			if test.wantSubstr != "" && !strings.Contains(err.Error(), test.wantSubstr) {
				t.Fatalf("Exchange() error = %q, want %q", err, test.wantSubstr)
			}
			for _, secret := range []string{"client-secret", "access-secret", "authorization-code", server.URL} {
				if strings.Contains(err.Error(), secret) {
					t.Fatalf("Exchange() exposed %q in error %q", secret, err)
				}
			}
		})
	}
}

func TestProviderUsesRequestContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		select {
		case <-request.Context().Done():
		case <-time.After(100 * time.Millisecond):
		}
		w.WriteHeader(http.StatusGatewayTimeout)
	}))
	defer server.Close()
	provider := NewProvider("client-id", "client-secret", "https://vigil.example.test/callback")
	provider.tokenEndpoint = server.URL + "/token"
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	_, err := provider.Exchange(ctx, "authorization-code", "verifier")
	if err != context.DeadlineExceeded {
		t.Fatalf("Exchange() error = %v, want context deadline", err)
	}
}
