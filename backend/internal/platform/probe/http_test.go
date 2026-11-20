package probe

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/endale-dev/vigil/internal/domain"
)

func TestHTTPProberStatusUserAgentAndProxySettings(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:1")
	t.Setenv("http_proxy", "http://127.0.0.1:1")
	type requestDetails struct {
		method    string
		userAgent string
	}
	requests := make(chan requestDetails, 3)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		requests <- requestDetails{method: request.Method, userAgent: request.UserAgent()}
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("response body is ignored"))
	}))
	defer server.Close()

	monitor := testHTTPMonitor(server.URL)
	monitor.ExpectedStatus = http.StatusAccepted
	result := newHTTPProber(true).Probe(context.Background(), monitor)
	if !result.Up || result.Detail != "HTTP 202" || result.Latency < 0 {
		t.Fatalf("Probe() = %+v, want successful HTTP 202 result", result)
	}
	request := <-requests
	if request.method != http.MethodGet || request.userAgent != "Vigil/1.0" {
		t.Fatalf("request = %+v, want GET with Vigil/1.0", request)
	}

	monitor.ExpectedStatus = 0
	result = newHTTPProber(true).Probe(context.Background(), monitor)
	if !result.Up {
		t.Fatalf("ExpectedStatus=0 rejected status below 400: %+v", result)
	}
	monitor.ExpectedStatus = http.StatusOK
	result = newHTTPProber(true).Probe(context.Background(), monitor)
	if result.Up {
		t.Fatalf("ExpectedStatus=200 accepted status 202: %+v", result)
	}
}

func TestHTTPProberClosesResponseWithoutReadingBody(t *testing.T) {
	requestCanceled := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		<-request.Context().Done()
		close(requestCanceled)
	}))
	defer server.Close()

	monitor := testHTTPMonitor(server.URL)
	monitor.Timeout = 5 * time.Second
	result := newHTTPProber(true).Probe(context.Background(), monitor)
	if !result.Up {
		t.Fatalf("Probe() = %+v, want successful response", result)
	}
	select {
	case <-requestCanceled:
	case <-time.After(time.Second):
		t.Fatal("response body was not closed promptly")
	}
}

func TestHTTPProberAllowsFiveRedirectsAndRejectsSixth(t *testing.T) {
	for _, test := range []struct {
		name       string
		redirects  int
		wantUp     bool
		wantDetail string
	}{
		{name: "five redirects", redirects: 5, wantUp: true, wantDetail: "HTTP 200"},
		{name: "six redirects", redirects: 6, wantDetail: "too many redirects"},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
				hop, _ := strconv.Atoi(strings.TrimPrefix(request.URL.Path, "/"))
				if hop < test.redirects {
					http.Redirect(w, request, fmt.Sprintf("/%d", hop+1), http.StatusFound)
					return
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			monitor := testHTTPMonitor(server.URL + "/0")
			result := newHTTPProber(true).Probe(context.Background(), monitor)
			if result.Up != test.wantUp || result.Detail != test.wantDetail {
				t.Fatalf("Probe() = %+v, want Up=%t Detail=%q", result, test.wantUp, test.wantDetail)
			}
		})
	}
}

func TestHTTPProberRejectsPrivateRedirect(t *testing.T) {
	dialer := &redirectTestDialer{}
	resolver := hostMapResolver{
		"safe.test":    {IP: net.ParseIP("8.8.8.8")},
		"private.test": {IP: net.ParseIP("10.0.0.1")},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		_, port, _ := net.SplitHostPort(request.Host)
		http.Redirect(w, request, "http://private.test:"+port+"/secret", http.StatusFound)
	}))
	defer server.Close()
	dialer.destination = server.Listener.Addr().String()
	guard := &addressGuard{resolver: resolver, dialer: dialer}
	_, port, err := net.SplitHostPort(server.Listener.Addr().String())
	if err != nil {
		t.Fatalf("split test server address: %v", err)
	}
	monitor := testHTTPMonitor("http://safe.test:" + port)

	result := (&httpProber{guard: guard}).Probe(context.Background(), monitor)
	if result.Up || result.Detail != "target address not allowed" {
		t.Fatalf("Probe() = %+v, want blocked private redirect", result)
	}
	if dialer.calls != 1 {
		t.Fatalf("dialer calls = %d, want only the initial destination", dialer.calls)
	}
	if strings.Contains(result.Detail, "private.test") || strings.Contains(result.Detail, "/secret") {
		t.Fatalf("detail exposed redirect destination: %q", result.Detail)
	}
}

func TestHTTPProberSanitizesTimeoutAndTransportErrors(t *testing.T) {
	t.Run("timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			<-request.Context().Done()
		}))
		defer server.Close()
		monitor := testHTTPMonitor(server.URL)
		monitor.Timeout = 50 * time.Millisecond
		result := newHTTPProber(true).Probe(context.Background(), monitor)
		if result.Up || result.Detail != "request timed out" {
			t.Fatalf("Probe() = %+v, want sanitized timeout", result)
		}
	})

	t.Run("connection failure", func(t *testing.T) {
		monitor := testHTTPMonitor("http://127.0.0.1:1/private?secret=value")
		result := newHTTPProber(true).Probe(context.Background(), monitor)
		if result.Up || result.Detail != "request failed" {
			t.Fatalf("Probe() = %+v, want sanitized connection failure", result)
		}
		if strings.Contains(result.Detail, "127.0.0.1") || strings.Contains(result.Detail, "secret") {
			t.Fatalf("detail exposed target information: %q", result.Detail)
		}
	})
}

func testHTTPMonitor(target string) domain.Monitor {
	return domain.Monitor{Target: target, Timeout: 2 * time.Second}
}

type hostMapResolver map[string]net.IPAddr

func (resolver hostMapResolver) LookupIPAddr(_ context.Context, host string) ([]net.IPAddr, error) {
	address, ok := resolver[host]
	if !ok {
		return nil, fmt.Errorf("unexpected hostname %q", host)
	}
	return []net.IPAddr{address}, nil
}

type redirectTestDialer struct {
	destination string
	calls       int
}

func (dialer *redirectTestDialer) DialContext(ctx context.Context, network, _ string) (net.Conn, error) {
	dialer.calls++
	return (&net.Dialer{}).DialContext(ctx, network, dialer.destination)
}
