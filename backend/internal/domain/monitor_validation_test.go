package domain

import (
	"strings"
	"testing"
	"time"
)

func validMonitor() Monitor {
	return Monitor{
		Name:     "API",
		Type:     MonitorHTTP,
		Target:   "https://example.com/health",
		Interval: time.Minute,
		Timeout:  10 * time.Second,
		Status:   StatusUnknown,
		Enabled:  true,
	}
}

func TestMonitorValidateAcceptsBoundaries(t *testing.T) {
	tests := []struct {
		name    string
		monitor Monitor
	}{
		{name: "minimum interval and timeout", monitor: func() Monitor { m := validMonitor(); m.Interval = 30 * time.Second; m.Timeout = time.Second; return m }()},
		{name: "maximum interval and timeout", monitor: func() Monitor {
			m := validMonitor()
			m.Interval = 24 * time.Hour
			m.Timeout = 30 * time.Second
			return m
		}()},
		{name: "maximum name length", monitor: func() Monitor { m := validMonitor(); m.Name = strings.Repeat("界", 80); return m }()},
		{name: "HTTP status boundaries", monitor: func() Monitor { m := validMonitor(); m.ExpectedStatus = 599; return m }()},
		{name: "TCP port boundaries", monitor: func() Monitor {
			m := validMonitor()
			m.Type = MonitorTCP
			m.Target = "example.com:1"
			m.ExpectedStatus = 0
			return m
		}()},
		{name: "TLS warning boundaries", monitor: func() Monitor {
			m := validMonitor()
			m.Type = MonitorTLS
			m.Target = "[2001:db8::1]:65535"
			m.TLSWarnDays = 90
			return m
		}()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := (&tt.monitor).Validate(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestMonitorValidateRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name  string
		edit  func(*Monitor)
		field string
	}{
		{name: "empty name", edit: func(m *Monitor) { m.Name = "  " }, field: "name"},
		{name: "long name", edit: func(m *Monitor) { m.Name = strings.Repeat("a", 81) }, field: "name"},
		{name: "unknown monitor type", edit: func(m *Monitor) { m.Type = "icmp" }, field: "type"},
		{name: "invalid status", edit: func(m *Monitor) { m.Status = "paused" }, field: "status"},
		{name: "short interval", edit: func(m *Monitor) { m.Interval = 29 * time.Second }, field: "interval"},
		{name: "long interval", edit: func(m *Monitor) { m.Interval = 24*time.Hour + time.Second }, field: "interval"},
		{name: "short timeout", edit: func(m *Monitor) { m.Timeout = time.Second - 1 }, field: "timeout"},
		{name: "timeout exceeds interval", edit: func(m *Monitor) { m.Interval = 30 * time.Second; m.Timeout = 31 * time.Second }, field: "timeout"},
		{name: "relative HTTP URL", edit: func(m *Monitor) { m.Target = "/health" }, field: "target"},
		{name: "non-HTTP scheme", edit: func(m *Monitor) { m.Target = "ftp://example.com" }, field: "target"},
		{name: "HTTP userinfo", edit: func(m *Monitor) { m.Target = "https://user:pass@example.com" }, field: "target"},
		{name: "HTTP fragment", edit: func(m *Monitor) { m.Target = "https://example.com/#" }, field: "target"},
		{name: "invalid expected status", edit: func(m *Monitor) { m.ExpectedStatus = 600 }, field: "expectedStatus"},
		{name: "expected status on TCP", edit: func(m *Monitor) { m.Type = MonitorTCP; m.Target = "example.com:80"; m.ExpectedStatus = 200 }, field: "expectedStatus"},
		{name: "invalid TCP port", edit: func(m *Monitor) { m.Type = MonitorTCP; m.Target = "example.com:65536" }, field: "target"},
		{name: "non-numeric TCP port", edit: func(m *Monitor) { m.Type = MonitorTCP; m.Target = "example.com:http" }, field: "target"},
		{name: "TLS without warning days", edit: func(m *Monitor) { m.Type = MonitorTLS; m.Target = "example.com:443" }, field: "tlsWarnDays"},
		{name: "TLS warning days on HTTP", edit: func(m *Monitor) { m.TLSWarnDays = 14 }, field: "tlsWarnDays"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			monitor := validMonitor()
			tt.edit(&monitor)
			err := (&monitor).Validate()
			validationErr, ok := err.(ValidationError)
			if !ok || validationErr.Field != tt.field {
				t.Fatalf("Validate() error = %#v, want ValidationError for %q", err, tt.field)
			}
		})
	}
}
