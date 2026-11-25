package routes

import (
	"errors"
	"net/http"
)

type AuthenticationHandler interface {
	Capabilities(http.ResponseWriter, *http.Request)
	Register(http.ResponseWriter, *http.Request)
	Login(http.ResponseWriter, *http.Request)
	Logout(http.ResponseWriter, *http.Request)
	GoogleStart(http.ResponseWriter, *http.Request)
	GoogleCallback(http.ResponseWriter, *http.Request)
	ForgotPassword(http.ResponseWriter, *http.Request)
	ResetPassword(http.ResponseWriter, *http.Request)
	ChangePassword(http.ResponseWriter, *http.Request)
	Me(http.ResponseWriter, *http.Request)
}

type AccountHandler interface {
	UpdateProfile(http.ResponseWriter, *http.Request)
	UpdateNotifications(http.ResponseWriter, *http.Request)
	UpdateStatusSlug(http.ResponseWriter, *http.Request)
	SendTestAlert(http.ResponseWriter, *http.Request)
}

type MonitorHandler interface {
	List(http.ResponseWriter, *http.Request)
	Create(http.ResponseWriter, *http.Request)
	Get(http.ResponseWriter, *http.Request)
	Update(http.ResponseWriter, *http.Request)
	Delete(http.ResponseWriter, *http.Request)
	Pause(http.ResponseWriter, *http.Request)
	Resume(http.ResponseWriter, *http.Request)
	Checks(http.ResponseWriter, *http.Request)
	Incidents(http.ResponseWriter, *http.Request)
}

type PublicStatusHandler interface {
	Get(http.ResponseWriter, *http.Request)
}

type HealthHandler interface {
	Get(http.ResponseWriter, *http.Request)
}

type Handlers struct {
	Authentication AuthenticationHandler
	Account        AccountHandler
	Monitors       MonitorHandler
	PublicStatus   PublicStatusHandler
	Health         HealthHandler
}

func New(handlers Handlers, requireAuth func(http.Handler) http.Handler) (*http.ServeMux, error) {
	if handlers.Authentication == nil || handlers.Account == nil || handlers.Monitors == nil || handlers.PublicStatus == nil || handlers.Health == nil || requireAuth == nil {
		return nil, errors.New("route dependencies are incomplete")
	}

	routes := []struct {
		pattern       string
		handler       http.Handler
		authenticated bool
	}{
		{"GET /api/auth/capabilities", http.HandlerFunc(handlers.Authentication.Capabilities), false},
		{"POST /api/auth/register", http.HandlerFunc(handlers.Authentication.Register), false},
		{"POST /api/auth/login", http.HandlerFunc(handlers.Authentication.Login), false},
		{"POST /api/auth/logout", http.HandlerFunc(handlers.Authentication.Logout), false},
		{"GET /api/auth/google/start", http.HandlerFunc(handlers.Authentication.GoogleStart), false},
		{"GET /api/auth/google/callback", http.HandlerFunc(handlers.Authentication.GoogleCallback), false},
		{"POST /api/auth/password/forgot", http.HandlerFunc(handlers.Authentication.ForgotPassword), false},
		{"POST /api/auth/password/reset", http.HandlerFunc(handlers.Authentication.ResetPassword), false},
		{"POST /api/auth/password/change", http.HandlerFunc(handlers.Authentication.ChangePassword), true},
		{"GET /api/me", http.HandlerFunc(handlers.Authentication.Me), true},
		{"PUT /api/me/profile", http.HandlerFunc(handlers.Account.UpdateProfile), true},
		{"PUT /api/me/notifications", http.HandlerFunc(handlers.Account.UpdateNotifications), true},
		{"PUT /api/me/status-slug", http.HandlerFunc(handlers.Account.UpdateStatusSlug), true},
		{"POST /api/me/notifications/test/{channel}", http.HandlerFunc(handlers.Account.SendTestAlert), true},
		{"GET /api/monitors", http.HandlerFunc(handlers.Monitors.List), true},
		{"POST /api/monitors", http.HandlerFunc(handlers.Monitors.Create), true},
		{"GET /api/monitors/{id}", http.HandlerFunc(handlers.Monitors.Get), true},
		{"PUT /api/monitors/{id}", http.HandlerFunc(handlers.Monitors.Update), true},
		{"DELETE /api/monitors/{id}", http.HandlerFunc(handlers.Monitors.Delete), true},
		{"POST /api/monitors/{id}/pause", http.HandlerFunc(handlers.Monitors.Pause), true},
		{"POST /api/monitors/{id}/resume", http.HandlerFunc(handlers.Monitors.Resume), true},
		{"GET /api/monitors/{id}/checks", http.HandlerFunc(handlers.Monitors.Checks), true},
		{"GET /api/monitors/{id}/incidents", http.HandlerFunc(handlers.Monitors.Incidents), true},
		{"GET /api/public/status/{slug}", http.HandlerFunc(handlers.PublicStatus.Get), false},
		{"GET /healthz", http.HandlerFunc(handlers.Health.Get), false},
	}

	mux := http.NewServeMux()
	registered := make(map[string]struct{}, len(routes))
	for _, route := range routes {
		if _, exists := registered[route.pattern]; exists {
			return nil, errors.New("duplicate route pattern")
		}
		registered[route.pattern] = struct{}{}
		if route.authenticated {
			route.handler = requireAuth(route.handler)
		}
		mux.Handle(route.pattern, route.handler)
	}
	return mux, nil
}
