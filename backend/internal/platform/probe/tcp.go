package probe

import (
	"context"
	"errors"
	"time"

	"github.com/endale-dev/vigil/internal/domain"
	"github.com/endale-dev/vigil/internal/service"
)

type tcpProber struct {
	guard *addressGuard
}

func newTCPProber(allowPrivate bool) *tcpProber {
	return &tcpProber{guard: newAddressGuard(allowPrivate)}
}

func (prober *tcpProber) Probe(ctx context.Context, monitor domain.Monitor) service.ProbeResult {
	requestCtx, cancel := context.WithTimeout(ctx, monitor.Timeout)
	defer cancel()
	startedAt := time.Now()
	connection, err := prober.guard.dialContext(requestCtx, "tcp", monitor.Target)
	latency := time.Since(startedAt)
	if err != nil {
		return service.ProbeResult{Latency: latency, Detail: safeTCPError(err, requestCtx)}
	}
	_ = connection.Close()
	return service.ProbeResult{Up: true, Latency: latency, Detail: "connected"}
}

func safeTCPError(err error, requestCtx context.Context) string {
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
