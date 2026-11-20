package probe

import (
	"context"

	"github.com/endale-dev/vigil/internal/domain"
	"github.com/endale-dev/vigil/internal/service"
)

type dispatcher struct {
	http *httpProber
	tcp  *tcpProber
	tls  *tlsProber
}

func New(allowPrivate bool, clock service.Clock) service.Prober {
	guard := newAddressGuard(allowPrivate)
	return &dispatcher{
		http: &httpProber{guard: guard},
		tcp:  &tcpProber{guard: guard},
		tls:  &tlsProber{guard: guard, clock: clock},
	}
}

func (dispatcher *dispatcher) Probe(ctx context.Context, monitor domain.Monitor) service.ProbeResult {
	switch monitor.Type {
	case domain.MonitorHTTP:
		return dispatcher.http.Probe(ctx, monitor)
	case domain.MonitorTCP:
		return dispatcher.tcp.Probe(ctx, monitor)
	case domain.MonitorTLS:
		return dispatcher.tls.Probe(ctx, monitor)
	default:
		return service.ProbeResult{Detail: "unsupported monitor type"}
	}
}

var _ service.Prober = (*dispatcher)(nil)
