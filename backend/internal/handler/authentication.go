package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/endale-dev/vigil/internal/domain"
	apperrors "github.com/endale-dev/vigil/internal/errors"
	"github.com/endale-dev/vigil/internal/service"
)

type authenticationService interface {
	Register(context.Context, string, string, string) (service.AuthenticationResult, error)
	Login(context.Context, string, string) (service.AuthenticationResult, error)
	Authenticate(context.Context, string) (service.AuthenticatedSession, error)
	Logout(context.Context, domain.ID) error
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
	cookie, err := request.Cookie(sessionCookieName)
	if err != nil {
		handler.writeError(writer, apperrors.ErrUnauthenticated)
		return
	}
	session, err := handler.authentication.Authenticate(request.Context(), cookie.Value)
	if err != nil {
		handler.writeError(writer, err)
		return
	}
	handler.writeJSON(writer, http.StatusOK, userDTO(
		session.User, session.HasPassword, handler.capabilities.Google, handler.capabilities.Telegram,
	))
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
