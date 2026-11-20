package probe

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"strconv"
	"time"
)

type ipResolver interface {
	LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error)
}

type contextDialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

type addressGuard struct {
	allowPrivate bool
	resolver     ipResolver
	dialer       contextDialer
}

func newAddressGuard(allowPrivate bool) *addressGuard {
	return &addressGuard{
		allowPrivate: allowPrivate,
		resolver:     net.DefaultResolver,
		dialer:       &net.Dialer{Timeout: 10 * time.Second},
	}
}

func (guard *addressGuard) dialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("invalid target address: %w", err)
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return nil, errors.New("invalid target port")
	}

	addresses, err := guard.resolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("resolve target: %w", err)
	}
	if len(addresses) == 0 {
		return nil, errors.New("target address not allowed")
	}
	for _, address := range addresses {
		if !guard.allowPrivate && isRestrictedAddress(address.IP) {
			return nil, errors.New("target address not allowed")
		}
		if _, ok := netip.AddrFromSlice(address.IP); !ok {
			return nil, errors.New("target address not allowed")
		}
	}

	var lastErr error
	for _, address := range addresses {
		conn, err := guard.dialer.DialContext(ctx, network, net.JoinHostPort(address.String(), port))
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

func isRestrictedAddress(ip net.IP) bool {
	address, ok := netip.AddrFromSlice(ip)
	if !ok {
		return true
	}
	address = address.Unmap()
	return address.IsLoopback() ||
		address.IsPrivate() ||
		address.IsLinkLocalUnicast() ||
		address.IsLinkLocalMulticast() ||
		address.IsUnspecified() ||
		address.IsMulticast()
}
