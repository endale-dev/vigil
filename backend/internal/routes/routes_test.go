package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type routeRecorder struct {
	calls map[string]int
}

func (recorder *routeRecorder) respond(name string, writer http.ResponseWriter) {
	recorder.calls[name]++
	writer.WriteHeader(http.StatusNoContent)
}

type authenticationRecorder struct{ *routeRecorder }

func (recorder authenticationRecorder) Capabilities(w http.ResponseWriter, _ *http.Request) {
	recorder.respond("capabilities", w)
}
func (recorder authenticationRecorder) Register(w http.ResponseWriter, _ *http.Request) {
	recorder.respond("register", w)
}
func (recorder authenticationRecorder) Login(w http.ResponseWriter, _ *http.Request) {
	recorder.respond("login", w)
}
func (recorder authenticationRecorder) Logout(w http.ResponseWriter, _ *http.Request) {
	recorder.respond("logout", w)
}
func (recorder authenticationRecorder) GoogleStart(w http.ResponseWriter, _ *http.Request) {
	recorder.respond("google_start", w)
}
func (recorder authenticationRecorder) GoogleCallback(w http.ResponseWriter, _ *http.Request) {
	recorder.respond("google_callback", w)
}
func (recorder authenticationRecorder) ForgotPassword(w http.ResponseWriter, _ *http.Request) {
	recorder.respond("forgot_password", w)
}
func (recorder authenticationRecorder) ResetPassword(w http.ResponseWriter, _ *http.Request) {
	recorder.respond("reset_password", w)
}
func (recorder authenticationRecorder) ChangePassword(w http.ResponseWriter, _ *http.Request) {
	recorder.respond("change_password", w)
}
func (recorder authenticationRecorder) Me(w http.ResponseWriter, _ *http.Request) {
	recorder.respond("me", w)
}

type accountRecorder struct{ *routeRecorder }

func (recorder accountRecorder) UpdateProfile(w http.ResponseWriter, _ *http.Request) {
	recorder.respond("update_profile", w)
}
func (recorder accountRecorder) UpdateNotifications(w http.ResponseWriter, _ *http.Request) {
	recorder.respond("update_notifications", w)
}
func (recorder accountRecorder) UpdateStatusSlug(w http.ResponseWriter, _ *http.Request) {
	recorder.respond("update_status_slug", w)
}
func (recorder accountRecorder) SendTestAlert(w http.ResponseWriter, _ *http.Request) {
	recorder.respond("test_alert", w)
}

type monitorRecorder struct{ *routeRecorder }

func (recorder monitorRecorder) List(w http.ResponseWriter, _ *http.Request) {
	recorder.respond("monitor_list", w)
}
func (recorder monitorRecorder) Create(w http.ResponseWriter, _ *http.Request) {
	recorder.respond("monitor_create", w)
}
func (recorder monitorRecorder) Get(w http.ResponseWriter, _ *http.Request) {
	recorder.respond("monitor_get", w)
}
func (recorder monitorRecorder) Update(w http.ResponseWriter, _ *http.Request) {
	recorder.respond("monitor_update", w)
}
func (recorder monitorRecorder) Delete(w http.ResponseWriter, _ *http.Request) {
	recorder.respond("monitor_delete", w)
}
func (recorder monitorRecorder) Pause(w http.ResponseWriter, _ *http.Request) {
	recorder.respond("monitor_pause", w)
}
func (recorder monitorRecorder) Resume(w http.ResponseWriter, _ *http.Request) {
	recorder.respond("monitor_resume", w)
}
func (recorder monitorRecorder) Checks(w http.ResponseWriter, _ *http.Request) {
	recorder.respond("monitor_checks", w)
}
func (recorder monitorRecorder) Incidents(w http.ResponseWriter, _ *http.Request) {
	recorder.respond("monitor_incidents", w)
}

type statusRecorder struct{ *routeRecorder }

func (recorder statusRecorder) Get(w http.ResponseWriter, _ *http.Request) {
	recorder.respond("public_status", w)
}

type healthRecorder struct{ *routeRecorder }

func (recorder healthRecorder) Get(w http.ResponseWriter, _ *http.Request) {
	recorder.respond("health", w)
}

func TestNewRegistersEveryAPIEndpointOnceAndProtectsOnlyRequiredRoutes(t *testing.T) {
	recorder := &routeRecorder{calls: make(map[string]int)}
	authenticated := 0
	requireAuth := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			authenticated++
			next.ServeHTTP(writer, request)
		})
	}
	router, err := New(Handlers{
		Authentication: authenticationRecorder{recorder},
		Account:        accountRecorder{recorder},
		Monitors:       monitorRecorder{recorder},
		PublicStatus:   statusRecorder{recorder},
		Health:         healthRecorder{recorder},
	}, requireAuth)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		method    string
		path      string
		handler   string
		protected bool
	}{
		{http.MethodGet, "/api/auth/capabilities", "capabilities", false},
		{http.MethodPost, "/api/auth/register", "register", false},
		{http.MethodPost, "/api/auth/login", "login", false},
		{http.MethodPost, "/api/auth/logout", "logout", false},
		{http.MethodGet, "/api/auth/google/start", "google_start", false},
		{http.MethodGet, "/api/auth/google/callback?code=code&state=state", "google_callback", false},
		{http.MethodPost, "/api/auth/password/forgot", "forgot_password", false},
		{http.MethodPost, "/api/auth/password/reset", "reset_password", false},
		{http.MethodPost, "/api/auth/password/change", "change_password", true},
		{http.MethodGet, "/api/me", "me", true},
		{http.MethodPut, "/api/me/profile", "update_profile", true},
		{http.MethodPut, "/api/me/notifications", "update_notifications", true},
		{http.MethodPut, "/api/me/status-slug", "update_status_slug", true},
		{http.MethodPost, "/api/me/notifications/test/email", "test_alert", true},
		{http.MethodGet, "/api/monitors", "monitor_list", true},
		{http.MethodPost, "/api/monitors", "monitor_create", true},
		{http.MethodGet, "/api/monitors/monitor-1", "monitor_get", true},
		{http.MethodPut, "/api/monitors/monitor-1", "monitor_update", true},
		{http.MethodDelete, "/api/monitors/monitor-1", "monitor_delete", true},
		{http.MethodPost, "/api/monitors/monitor-1/pause", "monitor_pause", true},
		{http.MethodPost, "/api/monitors/monitor-1/resume", "monitor_resume", true},
		{http.MethodGet, "/api/monitors/monitor-1/checks?limit=50", "monitor_checks", true},
		{http.MethodGet, "/api/monitors/monitor-1/incidents?limit=20", "monitor_incidents", true},
		{http.MethodGet, "/api/public/status/team", "public_status", false},
		{http.MethodGet, "/healthz", "health", false},
	}
	for _, test := range tests {
		request := httptest.NewRequest(test.method, test.path, nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusNoContent || recorder.calls[test.handler] != 1 {
			t.Errorf("%s %s returned %d and called %s %d times", test.method, request.URL.Path, response.Code, test.handler, recorder.calls[test.handler])
		}
		wantAuthCalls := 0
		if test.protected {
			wantAuthCalls = 1
		}
		if got := authenticated; got != wantAuthCalls {
			t.Errorf("%s %s passed through auth %d times, want %d", test.method, request.URL.Path, got, wantAuthCalls)
		}
		authenticated = 0
	}
	if len(tests) != 25 || len(recorder.calls) != 25 {
		t.Fatalf("registered %d handlers for %d endpoint checks, want 25", len(recorder.calls), len(tests))
	}
	for name, calls := range recorder.calls {
		if calls != 1 {
			t.Errorf("handler %s was called %d times, want once", name, calls)
		}
	}
}

func TestNewRejectsMissingDependencies(t *testing.T) {
	if router, err := New(Handlers{}, func(next http.Handler) http.Handler { return next }); err == nil || router != nil {
		t.Fatalf("New() = (%v, %v), want dependency error", router, err)
	}
	if router, err := New(Handlers{}, nil); err == nil || router != nil {
		t.Fatalf("New() = (%v, %v), want dependency error", router, err)
	}
}
