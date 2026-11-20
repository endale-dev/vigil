package probe

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/endale-dev/vigil/internal/domain"
	"github.com/endale-dev/vigil/internal/service"
)

const maxHTTPRedirects = 5

var errTooManyHTTPRedirects = errors.New("too many redirects")

type httpProber struct {
	guard *addressGuard
}

func newHTTPProber(allowPrivate bool) *httpProber {
	return &httpProber{guard: newAddressGuard(allowPrivate)}
}

func (prober *httpProber) Probe(ctx context.Context, monitor domain.Monitor) service.ProbeResult {
	requestCtx, cancel := context.WithTimeout(ctx, monitor.Timeout)
	defer cancel()

	request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, monitor.Target, nil)
	if err != nil {
		return service.ProbeResult{Detail: "invalid request"}
	}
	request.Header.Set("User-Agent", "Vigil/1.0")

	transport := &http.Transport{
		Proxy:       nil,
		DialContext: prober.guard.dialContext,
	}
	defer transport.CloseIdleConnections()
	client := &http.Client{
		Transport: transport,
		CheckRedirect: func(request *http.Request, via []*http.Request) error {
			if len(via) > maxHTTPRedirects {
				return errTooManyHTTPRedirects
			}
			if (request.URL.Scheme != "http" && request.URL.Scheme != "https") || request.URL.User != nil {
				return errTargetAddressNotAllowed
			}
			_, err := prober.guard.resolve(request.Context(), request.URL.Hostname())
			return err
		},
	}

	startedAt := time.Now()
	response, err := client.Do(request)
	latency := time.Since(startedAt)
	if err != nil {
		return service.ProbeResult{
			Latency: latency,
			Detail:  safeHTTPError(err, requestCtx),
		}
	}
	defer response.Body.Close()

	up := response.StatusCode == monitor.ExpectedStatus
	if monitor.ExpectedStatus == 0 {
		up = response.StatusCode < http.StatusBadRequest
	}
	return service.ProbeResult{
		Up:      up,
		Latency: latency,
		Detail:  fmt.Sprintf("HTTP %d", response.StatusCode),
	}
}

func safeHTTPError(err error, requestCtx context.Context) string {
	switch {
	case errors.Is(err, errTargetAddressNotAllowed):
		return "target address not allowed"
	case errors.Is(err, errTooManyHTTPRedirects):
		return "too many redirects"
	case errors.Is(err, context.DeadlineExceeded), errors.Is(requestCtx.Err(), context.DeadlineExceeded):
		return "request timed out"
	case errors.Is(err, context.Canceled), errors.Is(requestCtx.Err(), context.Canceled):
		return "request canceled"
	default:
		return "request failed"
	}
}
