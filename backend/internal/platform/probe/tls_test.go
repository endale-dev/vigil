package probe

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"testing"
	"time"

	"github.com/endale-dev/vigil/internal/domain"
)

func TestTLSProberReportsCertificateExpiryAndThreshold(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	server, roots, target := newTestTLSServer(t, now.Add(36*time.Hour))
	defer server.Close()

	prober := testTLSProber(now, roots)
	monitor := domain.Monitor{Target: target, Timeout: 2 * time.Second, TLSWarnDays: 3}
	result := prober.Probe(context.Background(), monitor)
	if result.Up || result.Detail != "certificate expires in 2 days" {
		t.Fatalf("Probe() = %+v, want a two-day warning", result)
	}

	monitor.TLSWarnDays = 1
	result = prober.Probe(context.Background(), monitor)
	if !result.Up || result.Detail != "certificate expires in 2 days" {
		t.Fatalf("Probe() = %+v, want valid certificate above threshold", result)
	}
}

func TestTLSProberRejectsExpiredAndUntrustedCertificates(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	t.Run("expired", func(t *testing.T) {
		server, roots, target := newTestTLSServer(t, now.Add(-time.Hour))
		defer server.Close()
		result := testTLSProber(now, roots).Probe(context.Background(), domain.Monitor{
			Target: target, Timeout: 2 * time.Second, TLSWarnDays: 1,
		})
		if result.Up || result.Detail != "certificate expired" {
			t.Fatalf("Probe() = %+v, want expired certificate detail", result)
		}
	})

	t.Run("untrusted", func(t *testing.T) {
		server, _, target := newTestTLSServer(t, now.Add(24*time.Hour))
		defer server.Close()
		result := testTLSProber(now, nil).Probe(context.Background(), domain.Monitor{
			Target: target, Timeout: 2 * time.Second, TLSWarnDays: 1,
		})
		if result.Up || result.Detail != "TLS connection failed" {
			t.Fatalf("Probe() = %+v, want untrusted certificate rejected", result)
		}
	})
}

func TestTLSProberUsesValidatedAddress(t *testing.T) {
	server, roots, target := newTestTLSServer(t, time.Now().Add(24*time.Hour))
	defer server.Close()
	listener := &recordingTLSDialer{destination: server.Addr().String()}
	prober := &tlsProber{
		guard: &addressGuard{
			allowPrivate: true,
			resolver:     staticResolver{ips: []net.IPAddr{{IP: net.ParseIP("127.0.0.1")}}},
			dialer:       listener,
		},
		clock: fixedProbeClock{now: time.Now()},
		roots: roots,
	}
	result := prober.Probe(context.Background(), domain.Monitor{Target: target, Timeout: 2 * time.Second, TLSWarnDays: 1})
	if !result.Up {
		t.Fatalf("Probe() = %+v, want successful verified TLS connection", result)
	}
	if listener.calls != 1 || listener.address != server.Addr().String() {
		t.Fatalf("TLS dial = (%d, %q), want one dial to validated server address %q", listener.calls, listener.address, server.Addr())
	}
}

func testTLSProber(now time.Time, roots *x509.CertPool) *tlsProber {
	return &tlsProber{
		guard: &addressGuard{
			allowPrivate: true,
			resolver:     staticResolver{ips: []net.IPAddr{{IP: net.ParseIP("127.0.0.1")}}},
			dialer:       &net.Dialer{},
		},
		clock: fixedProbeClock{now: now},
		roots: roots,
	}
}

type fixedProbeClock struct {
	now time.Time
}

func (clock fixedProbeClock) Now() time.Time {
	return clock.now
}

type recordingTLSDialer struct {
	destination string
	address     string
	calls       int
}

func (dialer *recordingTLSDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	dialer.calls++
	dialer.address = address
	return (&net.Dialer{}).DialContext(ctx, network, dialer.destination)
}

func newTestTLSServer(t *testing.T, notAfter time.Time) (net.Listener, *x509.CertPool, string) {
	t.Helper()
	now := time.Now()
	rootKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate root key: %v", err)
	}
	rootTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Vigil Test Root"},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(365 * 24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
	}
	rootDER, err := x509.CreateCertificate(rand.Reader, rootTemplate, rootTemplate, &rootKey.PublicKey, rootKey)
	if err != nil {
		t.Fatalf("create root certificate: %v", err)
	}
	root, err := x509.ParseCertificate(rootDER)
	if err != nil {
		t.Fatalf("parse root certificate: %v", err)
	}

	serverKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate server key: %v", err)
	}
	serverTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "example.test"},
		DNSNames:     []string{"example.test"},
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     notAfter,
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	serverDER, err := x509.CreateCertificate(rand.Reader, serverTemplate, root, &serverKey.PublicKey, rootKey)
	if err != nil {
		t.Fatalf("create server certificate: %v", err)
	}
	privateKey, err := x509.MarshalPKCS8PrivateKey(serverKey)
	if err != nil {
		t.Fatalf("marshal server key: %v", err)
	}
	certificate, err := tls.X509KeyPair(
		pemCertificate(serverDER),
		pemPrivateKey(privateKey),
	)
	if err != nil {
		t.Fatalf("load test key pair: %v", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen for TLS test server: %v", err)
	}
	tlsListener := tls.NewListener(listener, &tls.Config{Certificates: []tls.Certificate{certificate}})
	t.Cleanup(func() { _ = tlsListener.Close() })
	go func() {
		for {
			connection, err := tlsListener.Accept()
			if err != nil {
				return
			}
			go func(connection net.Conn) {
				defer connection.Close()
				if tlsConnection, ok := connection.(*tls.Conn); ok {
					_ = tlsConnection.Handshake()
				}
			}(connection)
		}
	}()

	roots := x509.NewCertPool()
	roots.AddCert(root)
	_, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("split TLS server address: %v", err)
	}
	return tlsListener, roots, net.JoinHostPort("example.test", port)
}

func pemCertificate(der []byte) []byte {
	return pemEncode("CERTIFICATE", der)
}

func pemPrivateKey(der []byte) []byte {
	return pemEncode("PRIVATE KEY", der)
}

func pemEncode(kind string, der []byte) []byte {
	return pem.EncodeToMemory(&pem.Block{Type: kind, Bytes: der})
}
