package handler

import (
	"errors"
	"log/slog"
	"net/http"

	apperrors "github.com/endale-dev/vigil/internal/errors"
)

func writeError(writer http.ResponseWriter, err error, logger *slog.Logger) error {
	var validation apperrors.ValidationError
	if errors.As(err, &validation) {
		return writeJSON(writer, http.StatusUnprocessableEntity, ErrorDTO{Error: validation.Message, Field: validation.Field})
	}

	status, message := http.StatusInternalServerError, "internal server error"
	switch {
	case errors.Is(err, errInvalidJSON):
		status, message = http.StatusBadRequest, "invalid JSON request"
	case errors.Is(err, errRequestTooLarge):
		status, message = http.StatusRequestEntityTooLarge, "request body is too large"
	case errors.Is(err, apperrors.ErrNotFound), errors.Is(err, apperrors.ErrProviderDisabled):
		status, message = http.StatusNotFound, "not found"
	case errors.Is(err, apperrors.ErrEmailTaken):
		status, message = http.StatusConflict, "email already taken"
	case errors.Is(err, apperrors.ErrSlugTaken):
		status, message = http.StatusConflict, "status slug already taken"
	case errors.Is(err, apperrors.ErrInvalidCredentials):
		status, message = http.StatusUnauthorized, "invalid credentials"
	case errors.Is(err, apperrors.ErrUnauthenticated):
		status, message = http.StatusUnauthorized, "unauthenticated"
	case errors.Is(err, apperrors.ErrInvalidToken):
		status, message = http.StatusBadRequest, "invalid token"
	case errors.Is(err, apperrors.ErrLimitReached):
		status, message = http.StatusForbidden, "limit reached"
	default:
		if logger != nil {
			logger.Error("request failed", "error", err)
		}
	}
	return writeJSON(writer, status, ErrorDTO{Error: message})
}
