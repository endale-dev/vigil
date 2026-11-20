package probe

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/endale-dev/vigil/internal/domain"
)

func TestDispatcherRoutesHTTPAndTCPMonitors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()
	accepted := make(chan struct{})
	go func() {
		connection, err := listener.Accept()
		if err == nil {
			_ = connection.Close()
			close(accepted)
		}
	}()

	now := time.Now()
	tlsServer, roots, tlsTarget := newTestTLSServer(t, now.Add(24*time.Hour))
	defer tlsServer.Close()
	prober := New(true, fixedProbeClock{now: now}).(*dispatcher)
	prober.http.guard.resolver = staticResolver{ips: []net.IPAddr{{IP: net.ParseIP("127.0.0.1")}}}
	prober.tls.roots = roots
	for _, monitor := range []domain.Monitor{
		{Type: domain.MonitorHTTP, Target: server.URL, Timeout: time.Second},
		{Type: domain.MonitorTCP, Target: listener.Addr().String(), Timeout: time.Second},
		{Type: domain.MonitorTLS, Target: tlsTarget, Timeout: time.Second, TLSWarnDays: 1},
	} {
		result := prober.Probe(context.Background(), monitor)
		if !result.Up {
			t.Fatalf("Probe(%s) = %+v, want up", monitor.Type, result)
		}
	}
	select {
	case <-accepted:
	case <-time.After(time.Second):
		t.Fatal("TCP dispatch did not connect to the local listener")
	}
}

func TestDispatcherRejectsUnknownMonitorType(t *testing.T) {
	result := New(false, fixedProbeClock{now: time.Now()}).Probe(context.Background(), domain.Monitor{
		Type:   "unknown",
		Target: "https://secret.example.test/path",
	})
	if result.Up || result.Detail != "unsupported monitor type" {
		t.Fatalf("Probe() = %+v, want safe unsupported-type result", result)
	}
}
