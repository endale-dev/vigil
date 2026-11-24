package handler

import (
	"context"
	"crypto/subtle"
	"errors"
	"log/slog"
	"net/http"

	"github.com/endale-dev/vigil/internal/domain"
	"github.com/endale-dev/vigil/internal/service"
)

type authenticationService interface {
	Register(context.Context, string, string, string) (service.AuthenticationResult, error)
	Login(context.Context, string, string) (service.AuthenticationResult, error)
	Authenticate(context.Context, string) (service.AuthenticatedSession, error)
	Logout(context.Context, domain.ID) error
	StartGoogleLogin() (service.GoogleLoginStart, error)
	GoogleCallback(context.Context, string, string) (service.AuthenticationResult, error)
	ForgotPassword(context.Context, string) error
	ResetPassword(context.Context, string, string) error
	ChangePassword(context.Context, domain.ID, domain.ID, string, string) error
}

type AuthenticationHandler struct {
	authentication authenticationService
	cookieSecure   bool
	capabilities   CapabilitiesDTO
	logger         *slog.Logger
}

type registerRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func NewAuthenticationHandler(authentication authenticationService, cookieSecure bool, capabilities CapabilitiesDTO, logger *slog.Logger) (*AuthenticationHandler, error) {
	if authentication == nil || logger == nil {
		return nil, errors.New("authentication handler dependencies are incomplete")
	}
	return &AuthenticationHandler{
		authentication: authentication,
		cookieSecure:   cookieSecure,
		capabilities:   capabilities,
		logger:         logger,
	}, nil
}

func (handler *AuthenticationHandler) Register(writer http.ResponseWriter, request *http.Request) {
	var input registerRequest
	if err := decodeJSON(writer, request, &input); err != nil {
		handler.writeError(writer, err)
		return
	}
	result, err := handler.authentication.Register(request.Context(), input.Name, input.Email, input.Password)
	if err != nil {
		handler.writeError(writer, err)
		return
	}
	setSessionCookie(writer, result.SessionToken, result.SessionExpiresAt, handler.cookieSecure)
	handler.writeJSON(writer, http.StatusCreated, userDTO(
		result.User, result.HasPassword, handler.capabilities.Google, handler.capabilities.Telegram,
	))
}

func (handler *AuthenticationHandler) Login(writer http.ResponseWriter, request *http.Request) {
	var input loginRequest
	if err := decodeJSON(writer, request, &input); err != nil {
		handler.writeError(writer, err)
		return
	}
	result, err := handler.authentication.Login(request.Context(), input.Email, input.Password)
	if err != nil {
		handler.writeError(writer, err)
		return
	}
	setSessionCookie(writer, result.SessionToken, result.SessionExpiresAt, handler.cookieSecure)
	handler.writeJSON(writer, http.StatusOK, userDTO(
		result.User, result.HasPassword, handler.capabilities.Google, handler.capabilities.Telegram,
	))
}

func (handler *AuthenticationHandler) Capabilities(writer http.ResponseWriter, _ *http.Request) {
	handler.writeJSON(writer, http.StatusOK, handler.capabilities)
}

func (handler *AuthenticationHandler) Logout(writer http.ResponseWriter, request *http.Request) {
	cookie, err := request.Cookie(sessionCookieName)
	if err != nil {
		clearSessionCookie(writer, handler.cookieSecure)
		writer.WriteHeader(http.StatusNoContent)
		return
	}

	session, err := handler.authentication.Authenticate(request.Context(), cookie.Value)
	if err != nil {
		clearSessionCookie(writer, handler.cookieSecure)
		writer.WriteHeader(http.StatusNoContent)
		return
	}
	if err := handler.authentication.Logout(request.Context(), session.Session.ID); err != nil {
		handler.writeError(writer, err)
		return
	}
	clearSessionCookie(writer, handler.cookieSecure)
	writer.WriteHeader(http.StatusNoContent)
}

func (handler *AuthenticationHandler) Me(writer http.ResponseWriter, request *http.Request) {
	session, err := authenticatedSession(request, handler.authentication)
	if err != nil {
		handler.writeError(writer, err)
		return
	}
	handler.writeJSON(writer, http.StatusOK, userDTO(
		session.User, session.HasPassword, handler.capabilities.Google, handler.capabilities.Telegram,
	))
}

func (handler *AuthenticationHandler) GoogleStart(writer http.ResponseWriter, request *http.Request) {
	login, err := handler.authentication.StartGoogleLogin()
	if err != nil {
		handler.writeError(writer, err)
		return
	}
	setOAuthCookies(writer, login.State, login.CodeVerifier, handler.cookieSecure)
	http.Redirect(writer, request, login.AuthorizationURL, http.StatusFound)
}

func (handler *AuthenticationHandler) GoogleCallback(writer http.ResponseWriter, request *http.Request) {
	stateCookie, stateErr := request.Cookie(oauthStateCookie)
	verifierCookie, verifierErr := request.Cookie(oauthVerifierCookie)
	code := request.URL.Query().Get("code")
	state := request.URL.Query().Get("state")
	if stateErr != nil || verifierErr != nil || code == "" || !validOAuthState(state, stateCookie.Value) {
		handler.redirectGoogleFailure(writer, request)
		return
	}

	result, err := handler.authentication.GoogleCallback(request.Context(), code, verifierCookie.Value)
	if err != nil {
		handler.redirectGoogleFailure(writer, request)
		return
	}
	clearOAuthCookies(writer, handler.cookieSecure)
	setSessionCookie(writer, result.SessionToken, result.SessionExpiresAt, handler.cookieSecure)
	http.Redirect(writer, request, "/", http.StatusFound)
}

func (handler *AuthenticationHandler) redirectGoogleFailure(writer http.ResponseWriter, request *http.Request) {
	clearOAuthCookies(writer, handler.cookieSecure)
	http.Redirect(writer, request, "/login?error=google", http.StatusFound)
}

func validOAuthState(query, cookie string) bool {
	return query != "" && cookie != "" && len(query) == len(cookie) && subtle.ConstantTimeCompare([]byte(query), []byte(cookie)) == 1
}

func (handler *AuthenticationHandler) writeError(writer http.ResponseWriter, err error) {
	if writeErr := writeError(writer, err, handler.logger); writeErr != nil {
		handler.logger.Error("could not write error response")
	}
}

func (handler *AuthenticationHandler) writeJSON(writer http.ResponseWriter, status int, value any) {
	if err := writeJSON(writer, status, value); err != nil {
		handler.logger.Error("could not write JSON response")
	}
}
