package handler

import (
	"context"
	"errors"
	"net/http"
	"time"
)

type mongoPinger interface {
	Ping(context.Context) error
}

type HealthHandler struct {
	pinger mongoPinger
}

func NewHealthHandler(pinger mongoPinger) (*HealthHandler, error) {
	if pinger == nil {
		return nil, errors.New("health handler pinger is required")
	}
	return &HealthHandler{pinger: pinger}, nil
}

func (handler *HealthHandler) Get(writer http.ResponseWriter, request *http.Request) {
	ctx, cancel := context.WithTimeout(request.Context(), 2*time.Second)
	defer cancel()
	if err := handler.pinger.Ping(ctx); err != nil {
		http.Error(writer, "unavailable", http.StatusServiceUnavailable)
		return
	}
	writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write([]byte("ok"))
}
