package handler

import (
	"net/http"

	apperrors "github.com/endale-dev/vigil/internal/errors"
)

type forgotPasswordRequest struct {
	Email string `json:"email"`
}

type resetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"newPassword"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

func (handler *AuthenticationHandler) ForgotPassword(writer http.ResponseWriter, request *http.Request) {
	var input forgotPasswordRequest
	if err := decodeJSON(writer, request, &input); err != nil {
		handler.writeError(writer, err)
		return
	}
	if err := handler.authentication.ForgotPassword(request.Context(), input.Email); err != nil {
		handler.writeError(writer, err)
		return
	}
	handler.writeJSON(writer, http.StatusAccepted, struct{}{})
}

func (handler *AuthenticationHandler) ResetPassword(writer http.ResponseWriter, request *http.Request) {
	var input resetPasswordRequest
	if err := decodeJSON(writer, request, &input); err != nil {
		handler.writeError(writer, err)
		return
	}
	if err := handler.authentication.ResetPassword(request.Context(), input.Token, input.NewPassword); err != nil {
		handler.writeError(writer, err)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

func (handler *AuthenticationHandler) ChangePassword(writer http.ResponseWriter, request *http.Request) {
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
	var input changePasswordRequest
	if err := decodeJSON(writer, request, &input); err != nil {
		handler.writeError(writer, err)
		return
	}
	if err := handler.authentication.ChangePassword(
		request.Context(), session.User.ID, session.Session.ID, input.CurrentPassword, input.NewPassword,
	); err != nil {
		handler.writeError(writer, err)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}
