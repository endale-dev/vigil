package handler

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/endale-dev/vigil/internal/domain"
	"github.com/endale-dev/vigil/internal/service"
)

type monitorServiceStub struct {
	listItems      []service.MonitorListItem
	listErr        error
	listOwner      domain.ID
	listCalls      int
	created        *domain.Monitor
	createErr      error
	createOwner    domain.ID
	createDraft    domain.Monitor
	createCalls    int
	monitor        *domain.Monitor
	getErr         error
	getOwner       domain.ID
	getID          domain.ID
	getCalls       int
	detail         service.MonitorDetail
	detailErr      error
	detailOwner    domain.ID
	detailID       domain.ID
	detailCalls    int
	updated        *domain.Monitor
	updateErr      error
	updateOwner    domain.ID
	updateID       domain.ID
	updateDraft    domain.Monitor
	updateCalls    int
	deleteErr      error
	deleteOwner    domain.ID
	deleteID       domain.ID
	deleteCalls    int
	pauseErr       error
	pauseOwner     domain.ID
	pauseID        domain.ID
	pauseCalls     int
	resumeErr      error
	resumeOwner    domain.ID
	resumeID       domain.ID
	resumeCalls    int
	checks         []domain.Check
	checksErr      error
	checksOwner    domain.ID
	checksID       domain.ID
	checksLimit    int
	checksCalls    int
	incidents      []domain.Incident
	incidentsErr   error
	incidentsOwner domain.ID
	incidentsID    domain.ID
	incidentsLimit int
	incidentsCalls int
}

func (stub *monitorServiceStub) ListWithUptime(_ context.Context, owner domain.ID) ([]service.MonitorListItem, error) {
	stub.listCalls++
	stub.listOwner = owner
	return stub.listItems, stub.listErr
}

func (stub *monitorServiceStub) Create(_ context.Context, owner domain.ID, draft domain.Monitor) (*domain.Monitor, error) {
	stub.createCalls++
	stub.createOwner = owner
	stub.createDraft = draft
	return stub.created, stub.createErr
}

func (stub *monitorServiceStub) Get(_ context.Context, owner, id domain.ID) (*domain.Monitor, error) {
	stub.getCalls++
	stub.getOwner = owner
	stub.getID = id
	return stub.monitor, stub.getErr
}

func (stub *monitorServiceStub) GetWithUptime(_ context.Context, owner, id domain.ID) (service.MonitorDetail, error) {
	stub.detailCalls++
	stub.detailOwner = owner
	stub.detailID = id
	return stub.detail, stub.detailErr
}

func (stub *monitorServiceStub) Update(_ context.Context, owner, id domain.ID, draft domain.Monitor) (*domain.Monitor, error) {
	stub.updateCalls++
	stub.updateOwner = owner
	stub.updateID = id
	stub.updateDraft = draft
	return stub.updated, stub.updateErr
}

func (stub *monitorServiceStub) Delete(_ context.Context, owner, id domain.ID) error {
	stub.deleteCalls++
	stub.deleteOwner = owner
	stub.deleteID = id
	return stub.deleteErr
}

func (stub *monitorServiceStub) Pause(_ context.Context, owner, id domain.ID) error {
	stub.pauseCalls++
	stub.pauseOwner = owner
	stub.pauseID = id
	return stub.pauseErr
}

func (stub *monitorServiceStub) Resume(_ context.Context, owner, id domain.ID) error {
	stub.resumeCalls++
	stub.resumeOwner = owner
	stub.resumeID = id
	return stub.resumeErr
}

func (stub *monitorServiceStub) ListChecks(_ context.Context, owner, id domain.ID, limit int) ([]domain.Check, error) {
	stub.checksCalls++
	stub.checksOwner = owner
	stub.checksID = id
	stub.checksLimit = limit
	return stub.checks, stub.checksErr
}

func (stub *monitorServiceStub) ListIncidents(_ context.Context, owner, id domain.ID, limit int) ([]domain.Incident, error) {
	stub.incidentsCalls++
	stub.incidentsOwner = owner
	stub.incidentsID = id
	stub.incidentsLimit = limit
	return stub.incidents, stub.incidentsErr
}

func TestMonitorListHandlerReturnsOwnedMonitorsWithDailyUptime(t *testing.T) {
	monitor := domain.Monitor{ID: "monitor-1", OwnerID: "owner-private", Name: "API", Type: domain.MonitorHTTP, Status: domain.StatusUp}
	monitorService := &monitorServiceStub{listItems: []service.MonitorListItem{{Monitor: monitor, Uptime: service.UptimeSummary{Up: 23, Total: 24}}}}
	handler := newMonitorHandlerForTest(t, monitorService, &authenticationStub{authResult: service.AuthenticatedSession{User: domain.User{ID: "owner-1"}}})
	request := httptest.NewRequest(http.MethodGet, "/api/monitors", nil)
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "session-token"})
	response := httptest.NewRecorder()
	handler.List(response, request)

	if response.Code != http.StatusOK || monitorService.listCalls != 1 || monitorService.listOwner != "owner-1" {
		t.Fatalf("list status/service call = %d/%+v", response.Code, monitorService)
	}
	var items []MonitorListDTO
	if err := json.Unmarshal(response.Body.Bytes(), &items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != "monitor-1" || items[0].Status != string(domain.StatusUp) || items[0].Uptime24Hours.Total != 24 {
		t.Fatalf("unexpected monitor list response: %+v", items)
	}
	if strings.Contains(response.Body.String(), "owner-private") || strings.Contains(response.Body.String(), "ownerId") {
		t.Fatalf("monitor list exposed ownership: %s", response.Body.String())
	}
}

func TestCreateMonitorHandlerMapsRequestAndReturnsCreatedDTO(t *testing.T) {
	monitorService := &monitorServiceStub{created: &domain.Monitor{
		ID: "monitor-1", OwnerID: "owner-1", Name: "API", Type: domain.MonitorHTTP,
		Target: "https://api.example", Interval: 90 * time.Second, Timeout: 5 * time.Second,
		Status: domain.StatusUnknown, Enabled: true,
	}}
	handler := newMonitorHandlerForTest(t, monitorService, &authenticationStub{authResult: service.AuthenticatedSession{User: domain.User{ID: "owner-1"}}})
	request := httptest.NewRequest(http.MethodPost, "/api/monitors", strings.NewReader(`{"name":"API","type":"http","target":"https://api.example","intervalSeconds":90,"timeoutSeconds":5,"expectedStatus":204,"public":true}`))
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "session-token"})
	response := httptest.NewRecorder()
	handler.Create(response, request)

	if response.Code != http.StatusCreated || monitorService.createCalls != 1 || monitorService.createOwner != "owner-1" {
		t.Fatalf("create status/service call = %d/%+v", response.Code, monitorService)
	}
	draft := monitorService.createDraft
	if draft.Name != "API" || draft.Type != domain.MonitorHTTP || draft.Target != "https://api.example" || draft.Interval != 90*time.Second || draft.Timeout != 5*time.Second || draft.ExpectedStatus != 204 || !draft.Public {
		t.Fatalf("created monitor draft = %+v", draft)
	}
	var result MonitorDTO
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.ID != "monitor-1" || result.IntervalSeconds != 90 || result.TimeoutSeconds != 5 {
		t.Fatalf("created monitor DTO = %+v", result)
	}
}

func TestCreateMonitorHandlerRejectsInvalidAndOversizedJSON(t *testing.T) {
	for _, test := range []struct {
		name       string
		body       string
		wantStatus int
	}{
		{name: "invalid JSON", body: `{"name":`, wantStatus: http.StatusBadRequest},
		{name: "body over limit", body: strings.Repeat(" ", maxJSONBodySize+1), wantStatus: http.StatusRequestEntityTooLarge},
	} {
		t.Run(test.name, func(t *testing.T) {
			monitorService := &monitorServiceStub{}
			handler := newMonitorHandlerForTest(t, monitorService, &authenticationStub{authResult: service.AuthenticatedSession{User: domain.User{ID: "owner-1"}}})
			request := httptest.NewRequest(http.MethodPost, "/api/monitors", strings.NewReader(test.body))
			request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "session-token"})
			response := httptest.NewRecorder()
			handler.Create(response, request)

			if response.Code != test.wantStatus || monitorService.createCalls != 0 {
				t.Fatalf("response/service calls = %d/%d, want %d/0", response.Code, monitorService.createCalls, test.wantStatus)
			}
		})
	}
}

func TestGetMonitorHandlerReturnsUptimeDetail(t *testing.T) {
	monitorService := &monitorServiceStub{detail: service.MonitorDetail{
		Monitor: domain.Monitor{ID: "monitor-1", OwnerID: "owner-1", Name: "API"},
		Uptime:  service.MonitorUptime{Last24Hours: service.UptimeSummary{Up: 4, Total: 5}, Last7Days: service.UptimeSummary{Up: 6, Total: 7}, Last30Days: service.UptimeSummary{Up: 8, Total: 9}},
	}}
	handler := newMonitorHandlerForTest(t, monitorService, &authenticationStub{authResult: service.AuthenticatedSession{User: domain.User{ID: "owner-1"}}})
	request := httptest.NewRequest(http.MethodGet, "/api/monitors/monitor-1", nil)
	request.SetPathValue("id", "monitor-1")
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "session-token"})
	response := httptest.NewRecorder()
	handler.Get(response, request)
	if response.Code != http.StatusOK || monitorService.detailOwner != "owner-1" || monitorService.detailID != "monitor-1" {
		t.Fatalf("get status/service args = %d/%q/%q", response.Code, monitorService.detailOwner, monitorService.detailID)
	}
	var detail MonitorDetailDTO
	if err := json.Unmarshal(response.Body.Bytes(), &detail); err != nil {
		t.Fatal(err)
	}
	if detail.ID != "monitor-1" || detail.Uptime.Last24Hours.Total != 5 || detail.Uptime.Last7Days.Total != 7 || detail.Uptime.Last30Days.Total != 9 {
		t.Fatalf("monitor detail response = %+v", detail)
	}
}

func TestUpdateMonitorHandlerMapsRequest(t *testing.T) {
	monitorService := &monitorServiceStub{updated: &domain.Monitor{ID: "monitor-1", OwnerID: "owner-1", Name: "Updated API"}}
	handler := newMonitorHandlerForTest(t, monitorService, &authenticationStub{authResult: service.AuthenticatedSession{User: domain.User{ID: "owner-1"}}})
	request := httptest.NewRequest(http.MethodPut, "/api/monitors/monitor-1", strings.NewReader(`{"name":"Updated API","type":"tcp","target":"example.test:443","intervalSeconds":60,"timeoutSeconds":10,"public":false}`))
	request.SetPathValue("id", "monitor-1")
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "session-token"})
	response := httptest.NewRecorder()
	handler.Update(response, request)
	if response.Code != http.StatusOK || monitorService.updateOwner != "owner-1" || monitorService.updateID != "monitor-1" {
		t.Fatalf("update status/service args = %d/%q/%q", response.Code, monitorService.updateOwner, monitorService.updateID)
	}
	if monitorService.updateDraft.Type != domain.MonitorTCP || monitorService.updateDraft.Interval != time.Minute || monitorService.updateDraft.Timeout != 10*time.Second {
		t.Fatalf("updated monitor draft = %+v", monitorService.updateDraft)
	}
}

func TestDeleteMonitorHandlerReturnsNoContent(t *testing.T) {
	monitorService := &monitorServiceStub{}
	handler := newMonitorHandlerForTest(t, monitorService, &authenticationStub{authResult: service.AuthenticatedSession{User: domain.User{ID: "owner-1"}}})
	request := httptest.NewRequest(http.MethodDelete, "/api/monitors/monitor-1", nil)
	request.SetPathValue("id", "monitor-1")
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "session-token"})
	response := httptest.NewRecorder()
	handler.Delete(response, request)
	if response.Code != http.StatusNoContent || response.Body.Len() != 0 || monitorService.deleteOwner != "owner-1" || monitorService.deleteID != "monitor-1" {
		t.Fatalf("delete status/body/service args = %d/%q/%q/%q", response.Code, response.Body.String(), monitorService.deleteOwner, monitorService.deleteID)
	}
}

func TestPauseAndResumeMonitorHandlersReturnUpdatedMonitor(t *testing.T) {
	for _, test := range []struct {
		name        string
		handle      func(*MonitorHandler, http.ResponseWriter, *http.Request)
		monitor     domain.Monitor
		wantEnabled bool
	}{
		{name: "pause", handle: (*MonitorHandler).Pause, monitor: domain.Monitor{ID: "monitor-1", Enabled: false}},
		{name: "resume", handle: (*MonitorHandler).Resume, monitor: domain.Monitor{ID: "monitor-1", Enabled: true}, wantEnabled: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			monitorService := &monitorServiceStub{monitor: &test.monitor}
			handler := newMonitorHandlerForTest(t, monitorService, &authenticationStub{authResult: service.AuthenticatedSession{User: domain.User{ID: "owner-1"}}})
			request := httptest.NewRequest(http.MethodPost, "/api/monitors/monitor-1/"+test.name, nil)
			request.SetPathValue("id", "monitor-1")
			request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "session-token"})
			response := httptest.NewRecorder()
			test.handle(handler, response, request)

			if response.Code != http.StatusOK || monitorService.getOwner != "owner-1" || monitorService.getID != "monitor-1" {
				t.Fatalf("response/service lookup = %d/%q/%q", response.Code, monitorService.getOwner, monitorService.getID)
			}
			if test.wantEnabled {
				if monitorService.resumeCalls != 1 || monitorService.resumeOwner != "owner-1" || monitorService.resumeID != "monitor-1" || monitorService.pauseCalls != 0 {
					t.Fatalf("resume service args = %+v", monitorService)
				}
			} else if monitorService.pauseCalls != 1 || monitorService.pauseOwner != "owner-1" || monitorService.pauseID != "monitor-1" || monitorService.resumeCalls != 0 {
				t.Fatalf("pause service args = %+v", monitorService)
			}
			var result MonitorDTO
			if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if result.ID != "monitor-1" || result.Enabled != test.wantEnabled {
				t.Fatalf("updated monitor response = %+v", result)
			}
		})
	}
}

func TestMonitorChecksAndIncidentsHandlersReturnDTOs(t *testing.T) {
	monitorService := &monitorServiceStub{
		checks:    []domain.Check{{ID: "check-1", MonitorID: "monitor-1", CheckedAt: time.Date(2026, time.October, 1, 12, 0, 0, 0, time.UTC), Up: true, Latency: 1500 * time.Millisecond}},
		incidents: []domain.Incident{{ID: "incident-1", MonitorID: "monitor-1", Cause: "timeout"}},
	}
	handler := newMonitorHandlerForTest(t, monitorService, &authenticationStub{authResult: service.AuthenticatedSession{User: domain.User{ID: "owner-1"}}})

	checksRequest := httptest.NewRequest(http.MethodGet, "/api/monitors/monitor-1/checks?limit=12", nil)
	checksRequest.SetPathValue("id", "monitor-1")
	checksRequest.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "session-token"})
	checksResponse := httptest.NewRecorder()
	handler.Checks(checksResponse, checksRequest)
	if checksResponse.Code != http.StatusOK || monitorService.checksOwner != "owner-1" || monitorService.checksID != "monitor-1" || monitorService.checksLimit != 12 {
		t.Fatalf("checks response/service args = %d/%q/%q/%d", checksResponse.Code, monitorService.checksOwner, monitorService.checksID, monitorService.checksLimit)
	}
	var checks []CheckDTO
	if err := json.Unmarshal(checksResponse.Body.Bytes(), &checks); err != nil {
		t.Fatal(err)
	}
	if len(checks) != 1 || checks[0].ID != "check-1" || checks[0].LatencyMS != 1500 {
		t.Fatalf("checks response = %+v", checks)
	}

	incidentsRequest := httptest.NewRequest(http.MethodGet, "/api/monitors/monitor-1/incidents", nil)
	incidentsRequest.SetPathValue("id", "monitor-1")
	incidentsRequest.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "session-token"})
	incidentsResponse := httptest.NewRecorder()
	handler.Incidents(incidentsResponse, incidentsRequest)
	if incidentsResponse.Code != http.StatusOK || monitorService.incidentsOwner != "owner-1" || monitorService.incidentsID != "monitor-1" || monitorService.incidentsLimit != 0 {
		t.Fatalf("incidents response/service args = %d/%q/%q/%d", incidentsResponse.Code, monitorService.incidentsOwner, monitorService.incidentsID, monitorService.incidentsLimit)
	}
	var incidents []IncidentDTO
	if err := json.Unmarshal(incidentsResponse.Body.Bytes(), &incidents); err != nil {
		t.Fatal(err)
	}
	if len(incidents) != 1 || incidents[0].ID != "incident-1" || incidents[0].Cause != "timeout" {
		t.Fatalf("incidents response = %+v", incidents)
	}
}

func TestMonitorChecksHandlerRejectsInvalidLimit(t *testing.T) {
	monitorService := &monitorServiceStub{}
	handler := newMonitorHandlerForTest(t, monitorService, &authenticationStub{authResult: service.AuthenticatedSession{User: domain.User{ID: "owner-1"}}})
	request := httptest.NewRequest(http.MethodGet, "/api/monitors/monitor-1/checks?limit=many", nil)
	request.SetPathValue("id", "monitor-1")
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "session-token"})
	response := httptest.NewRecorder()
	handler.Checks(response, request)
	if response.Code != http.StatusUnprocessableEntity || monitorService.checksCalls != 0 {
		t.Fatalf("invalid limit status/service calls = %d/%d", response.Code, monitorService.checksCalls)
	}
}

func TestMonitorHandlerRejectsDurationOverflowBeforeServiceCall(t *testing.T) {
	monitorService := &monitorServiceStub{}
	handler := newMonitorHandlerForTest(t, monitorService, &authenticationStub{authResult: service.AuthenticatedSession{User: domain.User{ID: "owner-1"}}})
	request := httptest.NewRequest(http.MethodPost, "/api/monitors", strings.NewReader(`{"name":"API","type":"http","target":"https://api.example","intervalSeconds":9223372037,"timeoutSeconds":5}`))
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "session-token"})
	response := httptest.NewRecorder()
	handler.Create(response, request)
	if response.Code != http.StatusUnprocessableEntity || monitorService.createCalls != 0 {
		t.Fatalf("overflow duration response/calls = %d/%d", response.Code, monitorService.createCalls)
	}
}

func TestNewMonitorHandlerRequiresDependencies(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	if handler, err := NewMonitorHandler(nil, &monitorServiceStub{}, logger); err == nil || handler != nil {
		t.Fatalf("NewMonitorHandler() = (%v, %v), want dependency error", handler, err)
	}
	if handler, err := NewMonitorHandler(&authenticationStub{}, nil, logger); err == nil || handler != nil {
		t.Fatalf("NewMonitorHandler() = (%v, %v), want dependency error", handler, err)
	}
	if handler, err := NewMonitorHandler(&authenticationStub{}, &monitorServiceStub{}, nil); err == nil || handler != nil {
		t.Fatalf("NewMonitorHandler() = (%v, %v), want dependency error", handler, err)
	}
}

func newMonitorHandlerForTest(t *testing.T, monitorStub monitorService, authStub sessionAuthenticator) *MonitorHandler {
	t.Helper()
	handler, err := NewMonitorHandler(authStub, monitorStub, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	return handler
}
