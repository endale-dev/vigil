package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/endale-dev/vigil/internal/service"
)

type publicStatusService interface {
	Get(context.Context, string) (service.PublicStatus, error)
}

type PublicStatusHandler struct {
	status publicStatusService
	logger *slog.Logger
}

func NewPublicStatusHandler(status publicStatusService, logger *slog.Logger) (*PublicStatusHandler, error) {
	if status == nil || logger == nil {
		return nil, errors.New("public status handler dependencies are incomplete")
	}
	return &PublicStatusHandler{status: status, logger: logger}, nil
}

func (handler *PublicStatusHandler) Get(writer http.ResponseWriter, request *http.Request) {
	status, err := handler.status.Get(request.Context(), request.PathValue("slug"))
	if err != nil {
		if writeErr := writeError(writer, err, handler.logger); writeErr != nil {
			handler.logger.Error("could not write error response")
		}
		return
	}
	writer.Header().Set("Cache-Control", "public, max-age=15")
	if err := writeJSON(writer, http.StatusOK, publicStatusDTO(status)); err != nil {
		handler.logger.Error("could not write JSON response")
	}
}
