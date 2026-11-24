package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/endale-dev/vigil/internal/domain"
	"github.com/endale-dev/vigil/internal/service"
)

type sessionAuthenticator interface {
	Authenticate(context.Context, string) (service.AuthenticatedSession, error)
}

type accountService interface {
	UpdateNotifications(context.Context, domain.ID, domain.NotifySettings) (*domain.User, error)
	UpdateStatusSlug(context.Context, domain.ID, string) (*domain.User, error)
	SendTestAlert(context.Context, domain.ID, domain.Channel) error
}

type AccountHandler struct {
	authentication sessionAuthenticator
	account        accountService
	capabilities   CapabilitiesDTO
	logger         *slog.Logger
}

type statusSlugRequest struct {
	StatusSlug string `json:"statusSlug"`
}

func NewAccountHandler(authentication sessionAuthenticator, account accountService, capabilities CapabilitiesDTO, logger *slog.Logger) (*AccountHandler, error) {
	if authentication == nil || account == nil || logger == nil {
		return nil, errors.New("account handler dependencies are incomplete")
	}
	return &AccountHandler{authentication: authentication, account: account, capabilities: capabilities, logger: logger}, nil
}

func (handler *AccountHandler) UpdateNotifications(writer http.ResponseWriter, request *http.Request) {
	session, ok := handler.currentSession(writer, request)
	if !ok {
		return
	}
	var input NotifyDTO
	if err := decodeJSON(writer, request, &input); err != nil {
		handler.writeError(writer, err)
		return
	}
	user, err := handler.account.UpdateNotifications(request.Context(), session.User.ID, domain.NotifySettings{
		EmailAlerts: input.EmailAlerts, TelegramChatID: input.TelegramChatID,
	})
	if err != nil {
		handler.writeError(writer, err)
		return
	}
	handler.writeJSON(writer, http.StatusOK, userDTO(*user, session.HasPassword, handler.capabilities.Google, handler.capabilities.Telegram))
}

func (handler *AccountHandler) UpdateStatusSlug(writer http.ResponseWriter, request *http.Request) {
	session, ok := handler.currentSession(writer, request)
	if !ok {
		return
	}
	var input statusSlugRequest
	if err := decodeJSON(writer, request, &input); err != nil {
		handler.writeError(writer, err)
		return
	}
	user, err := handler.account.UpdateStatusSlug(request.Context(), session.User.ID, input.StatusSlug)
	if err != nil {
		handler.writeError(writer, err)
		return
	}
	handler.writeJSON(writer, http.StatusOK, userDTO(*user, session.HasPassword, handler.capabilities.Google, handler.capabilities.Telegram))
}

func (handler *AccountHandler) SendTestAlert(writer http.ResponseWriter, request *http.Request) {
	session, ok := handler.currentSession(writer, request)
	if !ok {
		return
	}
	if err := handler.account.SendTestAlert(request.Context(), session.User.ID, domain.Channel(request.PathValue("channel"))); err != nil {
		handler.writeError(writer, err)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

func (handler *AccountHandler) currentSession(writer http.ResponseWriter, request *http.Request) (service.AuthenticatedSession, bool) {
	session, err := authenticatedSession(request, handler.authentication)
	if err != nil {
		handler.writeError(writer, err)
		return service.AuthenticatedSession{}, false
	}
	return session, true
}

func (handler *AccountHandler) writeError(writer http.ResponseWriter, err error) {
	if writeErr := writeError(writer, err, handler.logger); writeErr != nil {
		handler.logger.Error("could not write error response")
	}
}

func (handler *AccountHandler) writeJSON(writer http.ResponseWriter, status int, value any) {
	if err := writeJSON(writer, status, value); err != nil {
		handler.logger.Error("could not write JSON response")
	}
}
