package domain

import "time"

type Incident struct {
	ID         ID
	MonitorID  ID
	StartedAt  time.Time
	ResolvedAt time.Time
	Cause      string
}
