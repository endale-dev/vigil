package probe

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"
)

func TestIsRestrictedAddress(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		want bool
	}{
		{name: "IPv4 loopback", ip: "127.0.0.1", want: true},
		{name: "IPv4 private 10/8", ip: "10.0.0.1", want: true},
		{name: "IPv4 private 172.16/12", ip: "172.16.0.1", want: true},
		{name: "IPv4 private 192.168/16", ip: "192.168.1.1", want: true},
		{name: "IPv4 link local", ip: "169.254.1.1", want: true},
		{name: "IPv4 unspecified", ip: "0.0.0.0", want: true},
		{name: "IPv4 multicast", ip: "224.0.0.1", want: true},
		{name: "IPv6 loopback", ip: "::1", want: true},
		{name: "IPv6 unique local", ip: "fd00::1", want: true},
		{name: "IPv6 link local", ip: "fe80::1", want: true},
		{name: "IPv6 unspecified", ip: "::", want: true},
		{name: "IPv6 multicast", ip: "ff02::1", want: true},
		{name: "mapped IPv4 loopback", ip: "::ffff:127.0.0.1", want: true},
		{name: "public IPv4", ip: "8.8.8.8"},
		{name: "public IPv6", ip: "2606:4700:4700::1111"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRestrictedAddress(net.ParseIP(tt.ip)); got != tt.want {
				t.Fatalf("isRestrictedAddress(%s) = %t, want %t", tt.ip, got, tt.want)
			}
		})
	}
	if !isRestrictedAddress(nil) {
		t.Fatal("isRestrictedAddress(nil) = false, want true")
	}
}

func TestAddressGuardValidatesAllAnswersBeforeDial(t *testing.T) {
	dialer := &recordingDialer{}
	guard := &addressGuard{
		resolver: staticResolver{ips: []net.IPAddr{
			{IP: net.ParseIP("8.8.8.8")},
			{IP: net.ParseIP("10.0.0.1")},
		}},
		dialer: dialer,
	}

	_, err := guard.dialContext(context.Background(), "tcp", "example.test:443")
	if err == nil || err.Error() != "target address not allowed" {
		t.Fatalf("dialContext() error = %v, want target address not allowed", err)
	}
	if len(dialer.addresses) != 0 {
		t.Fatalf("dialer called with %v before all DNS answers were validated", dialer.addresses)
	}
}

func TestAddressGuardDialsResolvedIP(t *testing.T) {
	dialer := &recordingDialer{}
	guard := &addressGuard{
		resolver: staticResolver{ips: []net.IPAddr{{IP: net.ParseIP("8.8.8.8")}}},
		dialer:   dialer,
	}
	connection, err := guard.dialContext(context.Background(), "tcp", "example.test:443")
	if err != nil {
		t.Fatalf("dialContext() error = %v", err)
	}
	connection.Close()
	if len(dialer.addresses) != 1 || dialer.addresses[0] != "8.8.8.8:443" {
		t.Fatalf("dialer addresses = %v, want [8.8.8.8:443]", dialer.addresses)
	}
}

func TestAddressGuardAllowsPrivateWhenConfigured(t *testing.T) {
	dialer := &recordingDialer{}
	guard := &addressGuard{
		allowPrivate: true,
		resolver:     staticResolver{ips: []net.IPAddr{{IP: net.ParseIP("127.0.0.1")}}},
		dialer:       dialer,
	}
	connection, err := guard.dialContext(context.Background(), "tcp", "localhost:80")
	if err != nil {
		t.Fatalf("dialContext() error = %v", err)
	}
	connection.Close()
	if len(dialer.addresses) != 1 || dialer.addresses[0] != "127.0.0.1:80" {
		t.Fatalf("dialer addresses = %v, want [127.0.0.1:80]", dialer.addresses)
	}
}

func TestAddressGuardRejectsInvalidAddressAndPort(t *testing.T) {
	guard := &addressGuard{
		resolver: staticResolver{ips: []net.IPAddr{{IP: net.ParseIP("8.8.8.8")}}},
		dialer:   &recordingDialer{},
	}
	for _, address := range []string{"example.test", "example.test:0", "example.test:65536"} {
		t.Run(address, func(t *testing.T) {
			if _, err := guard.dialContext(context.Background(), "tcp", address); err == nil {
				t.Fatalf("dialContext(%q) returned no error", address)
			}
		})
	}
}

type staticResolver struct {
	ips []net.IPAddr
	err error
}

func (resolver staticResolver) LookupIPAddr(context.Context, string) ([]net.IPAddr, error) {
	return resolver.ips, resolver.err
}

type recordingDialer struct {
	addresses []string
}

func (dialer *recordingDialer) DialContext(_ context.Context, _, address string) (net.Conn, error) {
	dialer.addresses = append(dialer.addresses, address)
	if strings.HasSuffix(address, ":0") {
		return nil, errors.New("invalid port")
	}
	client, server := net.Pipe()
	server.Close()
	return client, nil
}
