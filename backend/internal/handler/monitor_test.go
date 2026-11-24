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
	listItems   []service.MonitorListItem
	listErr     error
	listOwner   domain.ID
	listCalls   int
	created     *domain.Monitor
	createErr   error
	createOwner domain.ID
	createDraft domain.Monitor
	createCalls int
	detail      service.MonitorDetail
	detailErr   error
	detailOwner domain.ID
	detailID    domain.ID
	detailCalls int
	updated     *domain.Monitor
	updateErr   error
	updateOwner domain.ID
	updateID    domain.ID
	updateDraft domain.Monitor
	updateCalls int
	deleteErr   error
	deleteOwner domain.ID
	deleteID    domain.ID
	deleteCalls int
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
