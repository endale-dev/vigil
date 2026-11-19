package domain

import "time"

type Check struct {
	ID        ID
	MonitorID ID
	CheckedAt time.Time
	Up        bool
	Latency   time.Duration
	Detail    string
}

type Uptime struct {
	Up    int
	Total int
}
