package server

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/endale-dev/vigil/internal/handler"
	"github.com/endale-dev/vigil/internal/routes"
)

type integrationAuthentication struct{ routes.AuthenticationHandler }

func (integrationAuthentication) Login(writer http.ResponseWriter, _ *http.Request) {
	writer.WriteHeader(http.StatusNoContent)
}

func (integrationAuthentication) Me(writer http.ResponseWriter, _ *http.Request) {
	writer.WriteHeader(http.StatusNoContent)
}

type integrationAccount struct{ routes.AccountHandler }

func (integrationAccount) UpdateNotifications(writer http.ResponseWriter, _ *http.Request) {
	writer.WriteHeader(http.StatusNoContent)
}

type integrationMonitors struct{ routes.MonitorHandler }
type integrationPublicStatus struct{ routes.PublicStatusHandler }

type integrationPinger struct{ err error }

func (pinger integrationPinger) Ping(context.Context) error { return pinger.err }

func TestRunWithRoutesIntegratesAPIProtectionAndFrontend(t *testing.T) {
	health, err := handler.NewHealthHandler(integrationPinger{})
	if err != nil {
		t.Fatal(err)
	}
	handlers := routes.Handlers{
		Authentication: integrationAuthentication{},
		Account:        integrationAccount{},
		Monitors:       integrationMonitors{},
		PublicStatus:   integrationPublicStatus{},
		Health:         health,
	}
	requireAuth := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if request.Header.Get("Authorization") != "Bearer test-token" {
				http.Error(writer, "unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(writer, request)
		})
	}

	address, err := freeTestAddress()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	serverDone := make(chan error, 1)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	go func() {
		serverDone <- RunWithRoutes(ctx, address, "production", fstest.MapFS{
			"dist/index.html":    &fstest.MapFile{Data: []byte("<main>Vigil app</main>")},
			"dist/assets/app.js": &fstest.MapFile{Data: []byte("console.log('vigil')")},
		}, handlers, requireAuth, false, logger, nil)
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-serverDone:
			if err != nil {
				t.Errorf("server shutdown: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Error("server did not stop after cancellation")
		}
	})

	baseURL := "http://" + address
	client := &http.Client{Timeout: 2 * time.Second, Transport: &http.Transport{Proxy: nil}}
	waitForServer(t, client, baseURL+"/healthz")

	response := getTestResponse(t, client, http.MethodGet, baseURL+"/api/me", nil)
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthed protected route status = %d, want 401", response.StatusCode)
	}
	response.Body.Close()
	request, err := http.NewRequest(http.MethodGet, baseURL+"/api/me", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer test-token")
	response, err = client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusNoContent {
		t.Errorf("authenticated protected route status = %d, want 204", response.StatusCode)
	}
	response.Body.Close()

	request, err = http.NewRequest(http.MethodPut, baseURL+"/api/me/notifications", strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Origin", "http://attacker.test")
	request.Header.Set("Sec-Fetch-Site", "cross-site")
	request.AddCookie(&http.Cookie{Name: "vigil_session", Value: "session-token"})
	response, err = client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusForbidden {
		t.Errorf("cross-origin state change status = %d, want 403", response.StatusCode)
	}
	response.Body.Close()

	for attempt := 1; attempt <= loginRateLimit+1; attempt++ {
		response = getTestResponse(t, client, http.MethodPost, baseURL+"/api/auth/login", nil)
		want := http.StatusNoContent
		if attempt == loginRateLimit+1 {
			want = http.StatusTooManyRequests
		}
		if response.StatusCode != want {
			t.Errorf("login attempt %d status = %d, want %d", attempt, response.StatusCode, want)
		}
		response.Body.Close()
	}

	response = getTestResponse(t, client, http.MethodGet, baseURL+"/status/team", nil)
	body, readErr := io.ReadAll(response.Body)
	response.Body.Close()
	if readErr != nil {
		t.Fatal(readErr)
	}
	if response.StatusCode != http.StatusOK || response.Header.Get("Cache-Control") != "no-cache" || !strings.Contains(string(body), "Vigil app") {
		t.Errorf("SPA fallback = %d, cache %q, body %q", response.StatusCode, response.Header.Get("Cache-Control"), body)
	}

	response = getTestResponse(t, client, http.MethodGet, baseURL+"/assets/app.js", nil)
	body, readErr = io.ReadAll(response.Body)
	response.Body.Close()
	if readErr != nil {
		t.Fatal(readErr)
	}
	if response.StatusCode != http.StatusOK || response.Header.Get("Cache-Control") != "public, max-age=31536000, immutable" || !strings.Contains(string(body), "console.log") {
		t.Errorf("asset response = %d, cache %q, body %q", response.StatusCode, response.Header.Get("Cache-Control"), body)
	}

	response = getTestResponse(t, client, http.MethodGet, baseURL+"/healthz", nil)
	body, readErr = io.ReadAll(response.Body)
	response.Body.Close()
	if readErr != nil {
		t.Fatal(readErr)
	}
	if response.StatusCode != http.StatusOK || string(body) != "ok" || response.Header.Get("Cache-Control") != "" {
		t.Errorf("health route = %d, cache %q, body %q", response.StatusCode, response.Header.Get("Cache-Control"), body)
	}

	response = getTestResponse(t, client, http.MethodGet, baseURL+"/api/missing", nil)
	if response.StatusCode != http.StatusNotFound {
		t.Errorf("unknown API route status = %d, want 404", response.StatusCode)
	}
	response.Body.Close()
}

func freeTestAddress() (string, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		return "", err
	}
	return address, nil
}

func waitForServer(t *testing.T, client *http.Client, url string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		response, err := client.Get(url)
		if err == nil {
			response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return
			}
		}
		select {
		case <-time.After(10 * time.Millisecond):
		}
	}
	t.Fatal("HTTP server did not become ready")
}

func getTestResponse(t *testing.T, client *http.Client, method, url string, body io.Reader) *http.Response {
	t.Helper()
	request, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	return response
}
