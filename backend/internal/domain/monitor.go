package domain

import "time"

type MonitorType string

const (
	MonitorHTTP MonitorType = "http"
	MonitorTCP  MonitorType = "tcp"
	MonitorTLS  MonitorType = "tls"
)

type Status string

const (
	StatusUnknown Status = "unknown"
	StatusUp      Status = "up"
	StatusDown    Status = "down"
)

type Monitor struct {
	ID                  ID
	OwnerID             ID
	Name                string
	Type                MonitorType
	Target              string
	Interval            time.Duration
	Timeout             time.Duration
	ExpectedStatus      int
	TLSWarnDays         int
	Public              bool
	Enabled             bool
	Status              Status
	ConsecutiveFailures int
	LastCheckedAt       time.Time
	NextCheckAt         time.Time
}
