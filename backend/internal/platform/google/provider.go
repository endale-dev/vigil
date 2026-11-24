package google

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/endale-dev/vigil/internal/service"
)

const (
	googleAuthorizationEndpoint = "https://accounts.google.com/o/oauth2/v2/auth"
	googleTokenEndpoint         = "https://oauth2.googleapis.com/token"
	googleUserInfoEndpoint      = "https://openidconnect.googleapis.com/v1/userinfo"
)

type Provider struct {
	clientID              string
	clientSecret          string
	redirectURI           string
	client                *http.Client
	authorizationEndpoint string
	tokenEndpoint         string
	userInfoEndpoint      string
}

func NewProvider(clientID, clientSecret, redirectURI string) *Provider {
	return &Provider{
		clientID:              clientID,
		clientSecret:          clientSecret,
		redirectURI:           redirectURI,
		client:                &http.Client{Timeout: 10 * time.Second},
		authorizationEndpoint: googleAuthorizationEndpoint,
		tokenEndpoint:         googleTokenEndpoint,
		userInfoEndpoint:      googleUserInfoEndpoint,
	}
}

func (provider *Provider) AuthURL(state, codeChallenge string) string {
	endpoint, err := url.Parse(provider.authorizationEndpoint)
	if err != nil {
		return ""
	}
	query := endpoint.Query()
	query.Set("client_id", provider.clientID)
	query.Set("redirect_uri", provider.redirectURI)
	query.Set("response_type", "code")
	query.Set("scope", "openid email profile")
	query.Set("state", state)
	query.Set("code_challenge", codeChallenge)
	query.Set("code_challenge_method", "S256")
	endpoint.RawQuery = query.Encode()
	return endpoint.String()
}

func (provider *Provider) Exchange(ctx context.Context, code, codeVerifier string) (service.GoogleIdentity, error) {
	form := url.Values{
		"code":          {code},
		"client_id":     {provider.clientID},
		"client_secret": {provider.clientSecret},
		"redirect_uri":  {provider.redirectURI},
		"grant_type":    {"authorization_code"},
		"code_verifier": {codeVerifier},
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, provider.tokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return service.GoogleIdentity{}, errors.New("google token exchange failed")
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := provider.client.Do(request)
	if err != nil {
		if ctx.Err() != nil {
			return service.GoogleIdentity{}, ctx.Err()
		}
		return service.GoogleIdentity{}, errors.New("google token exchange failed")
	}
	var token struct {
		AccessToken string `json:"access_token"`
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices || decodeGoogleJSON(response.Body, &token) != nil || token.AccessToken == "" {
		_ = response.Body.Close()
		return service.GoogleIdentity{}, errors.New("google token exchange failed")
	}
	_ = response.Body.Close()

	request, err = http.NewRequestWithContext(ctx, http.MethodGet, provider.userInfoEndpoint, nil)
	if err != nil {
		return service.GoogleIdentity{}, errors.New("google identity lookup failed")
	}
	request.Header.Set("Authorization", "Bearer "+token.AccessToken)
	response, err = provider.client.Do(request)
	if err != nil {
		if ctx.Err() != nil {
			return service.GoogleIdentity{}, ctx.Err()
		}
		return service.GoogleIdentity{}, errors.New("google identity lookup failed")
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return service.GoogleIdentity{}, errors.New("google identity lookup failed")
	}
	var profile struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		Name          string `json:"name"`
		EmailVerified bool   `json:"email_verified"`
	}
	if err := decodeGoogleJSON(response.Body, &profile); err != nil {
		return service.GoogleIdentity{}, errors.New("google identity lookup failed")
	}
	if profile.Sub == "" || profile.Email == "" || !profile.EmailVerified {
		return service.GoogleIdentity{}, errors.New("google identity is not verified")
	}
	return service.GoogleIdentity{
		Sub:           profile.Sub,
		Email:         profile.Email,
		Name:          profile.Name,
		EmailVerified: profile.EmailVerified,
	}, nil
}

func decodeGoogleJSON(reader io.Reader, destination any) error {
	decoder := json.NewDecoder(io.LimitReader(reader, 1<<20))
	return decoder.Decode(destination)
}

var _ service.GoogleProvider = (*Provider)(nil)
