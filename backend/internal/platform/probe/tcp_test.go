package probe

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/endale-dev/vigil/internal/domain"
)

func TestTCPProberConnectsToLocalListenerWhenPrivateTargetsAreAllowed(t *testing.T) {
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

	monitor := domain.Monitor{Target: listener.Addr().String(), Timeout: time.Second}
	result := newTCPProber(true).Probe(context.Background(), monitor)
	if !result.Up || result.Detail != "connected" || result.Latency < 0 {
		t.Fatalf("Probe() = %+v, want successful TCP result", result)
	}
	select {
	case <-accepted:
	case <-time.After(time.Second):
		t.Fatal("local listener did not accept the probe connection")
	}
}

func TestTCPProberBlocksPrivateTargetsByDefault(t *testing.T) {
	dialer := &recordingDialer{}
	prober := &tcpProber{guard: &addressGuard{
		resolver: staticResolver{ips: []net.IPAddr{{IP: net.ParseIP("127.0.0.1")}}},
		dialer:   dialer,
	}}
	result := prober.Probe(context.Background(), domain.Monitor{Target: "localhost:80", Timeout: time.Second})
	if result.Up || result.Detail != "target address not allowed" {
		t.Fatalf("Probe() = %+v, want private target blocked", result)
	}
	if len(dialer.addresses) != 0 {
		t.Fatalf("dialer called for blocked target: %v", dialer.addresses)
	}
}
