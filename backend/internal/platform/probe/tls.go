package probe

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"math"
	"net"
	"time"

	"github.com/endale-dev/vigil/internal/domain"
	"github.com/endale-dev/vigil/internal/service"
)

type tlsProber struct {
	guard *addressGuard
	clock service.Clock
	roots *x509.CertPool
}

func (prober *tlsProber) Probe(ctx context.Context, monitor domain.Monitor) service.ProbeResult {
	host, port, err := net.SplitHostPort(monitor.Target)
	if err != nil || host == "" {
		return service.ProbeResult{Detail: "invalid target"}
	}
	requestCtx, cancel := context.WithTimeout(ctx, monitor.Timeout)
	defer cancel()

	startedAt := time.Now()
	connection, err := prober.guard.dialContext(requestCtx, "tcp", net.JoinHostPort(host, port))
	if err != nil {
		return service.ProbeResult{Latency: time.Since(startedAt), Detail: safeTLSNetworkError(err, requestCtx)}
	}
	defer connection.Close()

	tlsConnection := tls.Client(connection, &tls.Config{
		ServerName: host,
		RootCAs:    prober.roots,
	})
	if err := tlsConnection.HandshakeContext(requestCtx); err != nil {
		return service.ProbeResult{Latency: time.Since(startedAt), Detail: safeTLSHandshakeError(err, requestCtx)}
	}
	state := tlsConnection.ConnectionState()
	latency := time.Since(startedAt)
	if len(state.PeerCertificates) == 0 {
		return service.ProbeResult{Latency: latency, Detail: "TLS connection failed"}
	}

	remaining := state.PeerCertificates[0].NotAfter.Sub(prober.clock.Now())
	if remaining <= 0 {
		return service.ProbeResult{Latency: latency, Detail: "certificate expired"}
	}
	days := int(math.Ceil(remaining.Hours() / 24))
	return service.ProbeResult{
		Up:      days >= monitor.TLSWarnDays,
		Latency: latency,
		Detail:  fmt.Sprintf("certificate expires in %d days", days),
	}
}

func safeTLSNetworkError(err error, requestCtx context.Context) string {
	switch {
	case errors.Is(err, errTargetAddressNotAllowed):
		return "target address not allowed"
	case errors.Is(err, context.DeadlineExceeded), errors.Is(requestCtx.Err(), context.DeadlineExceeded):
		return "connection timed out"
	case errors.Is(err, context.Canceled), errors.Is(requestCtx.Err(), context.Canceled):
		return "connection canceled"
	default:
		return "connection failed"
	}
}

func safeTLSHandshakeError(err error, requestCtx context.Context) string {
	var invalidCertificate x509.CertificateInvalidError
	if errors.As(err, &invalidCertificate) && invalidCertificate.Reason == x509.Expired {
		return "certificate expired"
	}
	switch {
	case errors.Is(err, context.DeadlineExceeded), errors.Is(requestCtx.Err(), context.DeadlineExceeded):
		return "TLS handshake timed out"
	case errors.Is(err, context.Canceled), errors.Is(requestCtx.Err(), context.Canceled):
		return "TLS handshake canceled"
	default:
		return "TLS connection failed"
	}
}
