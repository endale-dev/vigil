package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/endale-dev/vigil/internal/domain"
	apperrors "github.com/endale-dev/vigil/internal/errors"
	"github.com/endale-dev/vigil/internal/service"
)

type monitorService interface {
	ListWithUptime(context.Context, domain.ID) ([]service.MonitorListItem, error)
	Create(context.Context, domain.ID, domain.Monitor) (*domain.Monitor, error)
	Get(context.Context, domain.ID, domain.ID) (*domain.Monitor, error)
	GetWithUptime(context.Context, domain.ID, domain.ID) (service.MonitorDetail, error)
	Update(context.Context, domain.ID, domain.ID, domain.Monitor) (*domain.Monitor, error)
	Delete(context.Context, domain.ID, domain.ID) error
	Pause(context.Context, domain.ID, domain.ID) error
	Resume(context.Context, domain.ID, domain.ID) error
	ListChecks(context.Context, domain.ID, domain.ID, int) ([]domain.Check, error)
	ListIncidents(context.Context, domain.ID, domain.ID, int) ([]domain.Incident, error)
}

type MonitorHandler struct {
	authentication sessionAuthenticator
	monitors       monitorService
	logger         *slog.Logger
}

type monitorRequest struct {
	Name            string             `json:"name"`
	Type            domain.MonitorType `json:"type"`
	Target          string             `json:"target"`
	IntervalSeconds int64              `json:"intervalSeconds"`
	TimeoutSeconds  int64              `json:"timeoutSeconds"`
	ExpectedStatus  int                `json:"expectedStatus"`
	TLSWarnDays     int                `json:"tlsWarnDays"`
	Public          bool               `json:"public"`
}

func NewMonitorHandler(authentication sessionAuthenticator, monitors monitorService, logger *slog.Logger) (*MonitorHandler, error) {
	if authentication == nil || monitors == nil || logger == nil {
		return nil, errors.New("monitor handler dependencies are incomplete")
	}
	return &MonitorHandler{authentication: authentication, monitors: monitors, logger: logger}, nil
}

func (handler *MonitorHandler) List(writer http.ResponseWriter, request *http.Request) {
	session, ok := handler.currentSession(writer, request)
	if !ok {
		return
	}
	items, err := handler.monitors.ListWithUptime(request.Context(), session.User.ID)
	if err != nil {
		handler.writeError(writer, err)
		return
	}
	response := make([]MonitorListDTO, 0, len(items))
	for _, item := range items {
		response = append(response, monitorListDTO(item))
	}
	handler.writeJSON(writer, http.StatusOK, response)
}

func (handler *MonitorHandler) Create(writer http.ResponseWriter, request *http.Request) {
	session, ok := handler.currentSession(writer, request)
	if !ok {
		return
	}
	var input monitorRequest
	if err := decodeJSON(writer, request, &input); err != nil {
		handler.writeError(writer, err)
		return
	}
	draft, err := input.domainMonitor()
	if err != nil {
		handler.writeError(writer, err)
		return
	}
	monitor, err := handler.monitors.Create(request.Context(), session.User.ID, draft)
	if err != nil {
		handler.writeError(writer, err)
		return
	}
	handler.writeJSON(writer, http.StatusCreated, monitorDTO(*monitor))
}

func (handler *MonitorHandler) Get(writer http.ResponseWriter, request *http.Request) {
	session, ok := handler.currentSession(writer, request)
	if !ok {
		return
	}
	detail, err := handler.monitors.GetWithUptime(request.Context(), session.User.ID, domain.ID(request.PathValue("id")))
	if err != nil {
		handler.writeError(writer, err)
		return
	}
	handler.writeJSON(writer, http.StatusOK, monitorDetailDTO(detail))
}

func (handler *MonitorHandler) Update(writer http.ResponseWriter, request *http.Request) {
	session, ok := handler.currentSession(writer, request)
	if !ok {
		return
	}
	var input monitorRequest
	if err := decodeJSON(writer, request, &input); err != nil {
		handler.writeError(writer, err)
		return
	}
	draft, err := input.domainMonitor()
	if err != nil {
		handler.writeError(writer, err)
		return
	}
	monitor, err := handler.monitors.Update(request.Context(), session.User.ID, domain.ID(request.PathValue("id")), draft)
	if err != nil {
		handler.writeError(writer, err)
		return
	}
	handler.writeJSON(writer, http.StatusOK, monitorDTO(*monitor))
}

func (handler *MonitorHandler) Delete(writer http.ResponseWriter, request *http.Request) {
	session, ok := handler.currentSession(writer, request)
	if !ok {
		return
	}
	if err := handler.monitors.Delete(request.Context(), session.User.ID, domain.ID(request.PathValue("id"))); err != nil {
		handler.writeError(writer, err)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

func (handler *MonitorHandler) Pause(writer http.ResponseWriter, request *http.Request) {
	handler.setEnabled(writer, request, false)
}

func (handler *MonitorHandler) Resume(writer http.ResponseWriter, request *http.Request) {
	handler.setEnabled(writer, request, true)
}

func (handler *MonitorHandler) Checks(writer http.ResponseWriter, request *http.Request) {
	session, ok := handler.currentSession(writer, request)
	if !ok {
		return
	}
	limit, err := queryLimit(request, 200)
	if err != nil {
		handler.writeError(writer, err)
		return
	}
	checks, err := handler.monitors.ListChecks(request.Context(), session.User.ID, domain.ID(request.PathValue("id")), limit)
	if err != nil {
		handler.writeError(writer, err)
		return
	}
	response := make([]CheckDTO, 0, len(checks))
	for _, check := range checks {
		response = append(response, checkDTO(check))
	}
	handler.writeJSON(writer, http.StatusOK, response)
}

func (handler *MonitorHandler) Incidents(writer http.ResponseWriter, request *http.Request) {
	session, ok := handler.currentSession(writer, request)
	if !ok {
		return
	}
	limit, err := queryLimit(request, 50)
	if err != nil {
		handler.writeError(writer, err)
		return
	}
	incidents, err := handler.monitors.ListIncidents(request.Context(), session.User.ID, domain.ID(request.PathValue("id")), limit)
	if err != nil {
		handler.writeError(writer, err)
		return
	}
	response := make([]IncidentDTO, 0, len(incidents))
	for _, incident := range incidents {
		response = append(response, incidentDTO(incident))
	}
	handler.writeJSON(writer, http.StatusOK, response)
}

func (handler *MonitorHandler) setEnabled(writer http.ResponseWriter, request *http.Request, enabled bool) {
	session, ok := handler.currentSession(writer, request)
	if !ok {
		return
	}
	monitorID := domain.ID(request.PathValue("id"))
	var err error
	if enabled {
		err = handler.monitors.Resume(request.Context(), session.User.ID, monitorID)
	} else {
		err = handler.monitors.Pause(request.Context(), session.User.ID, monitorID)
	}
	if err != nil {
		handler.writeError(writer, err)
		return
	}
	monitor, err := handler.monitors.Get(request.Context(), session.User.ID, monitorID)
	if err != nil {
		handler.writeError(writer, err)
		return
	}
	handler.writeJSON(writer, http.StatusOK, monitorDTO(*monitor))
}

func queryLimit(request *http.Request, maximum int) (int, error) {
	value := request.URL.Query().Get("limit")
	if value == "" {
		return 0, nil
	}
	limit, err := strconv.Atoi(value)
	if err != nil {
		return 0, apperrors.ValidationError{Field: "limit", Message: "must be an integer"}
	}
	if limit > maximum {
		return maximum, nil
	}
	return limit, nil
}

func (input monitorRequest) domainMonitor() (domain.Monitor, error) {
	interval, err := durationFromSeconds(input.IntervalSeconds, "intervalSeconds")
	if err != nil {
		return domain.Monitor{}, err
	}
	timeout, err := durationFromSeconds(input.TimeoutSeconds, "timeoutSeconds")
	if err != nil {
		return domain.Monitor{}, err
	}
	return domain.Monitor{
		Name: input.Name, Type: input.Type, Target: input.Target,
		Interval: interval, Timeout: timeout, ExpectedStatus: input.ExpectedStatus,
		TLSWarnDays: input.TLSWarnDays, Public: input.Public,
	}, nil
}

func durationFromSeconds(seconds int64, field string) (time.Duration, error) {
	maxSeconds := int64((1<<63 - 1) / int64(time.Second))
	if seconds > maxSeconds || seconds < -maxSeconds {
		return 0, apperrors.ValidationError{Field: field, Message: "is out of range"}
	}
	return time.Duration(seconds) * time.Second, nil
}

func (handler *MonitorHandler) currentSession(writer http.ResponseWriter, request *http.Request) (service.AuthenticatedSession, bool) {
	cookie, err := request.Cookie(sessionCookieName)
	if err != nil {
		handler.writeError(writer, apperrors.ErrUnauthenticated)
		return service.AuthenticatedSession{}, false
	}
	session, err := handler.authentication.Authenticate(request.Context(), cookie.Value)
	if err != nil {
		handler.writeError(writer, err)
		return service.AuthenticatedSession{}, false
	}
	return session, true
}

func (handler *MonitorHandler) writeError(writer http.ResponseWriter, err error) {
	if writeErr := writeError(writer, err, handler.logger); writeErr != nil {
		handler.logger.Error("could not write error response")
	}
}

func (handler *MonitorHandler) writeJSON(writer http.ResponseWriter, status int, value any) {
	if err := writeJSON(writer, status, value); err != nil {
		handler.logger.Error("could not write JSON response")
	}
}
