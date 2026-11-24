package handler

import (
	"time"

	"github.com/endale-dev/vigil/internal/domain"
	"github.com/endale-dev/vigil/internal/service"
)

type UserDTO struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Email        string          `json:"email"`
	HasPassword  bool            `json:"hasPassword"`
	GoogleLinked bool            `json:"googleLinked"`
	StatusSlug   string          `json:"statusSlug"`
	Notify       NotifyDTO       `json:"notify"`
	Capabilities CapabilitiesDTO `json:"capabilities"`
}

type NotifyDTO struct {
	EmailAlerts    bool   `json:"emailAlerts"`
	TelegramChatID string `json:"telegramChatId"`
}

type CapabilitiesDTO struct {
	Google   bool `json:"google"`
	Telegram bool `json:"telegram"`
}

type MonitorDTO struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	Type            string     `json:"type"`
	Target          string     `json:"target"`
	IntervalSeconds float64    `json:"intervalSeconds"`
	TimeoutSeconds  float64    `json:"timeoutSeconds"`
	ExpectedStatus  int        `json:"expectedStatus"`
	TLSWarnDays     int        `json:"tlsWarnDays"`
	Public          bool       `json:"public"`
	Enabled         bool       `json:"enabled"`
	Status          string     `json:"status"`
	LastCheckedAt   *time.Time `json:"lastCheckedAt"`
	CreatedAt       time.Time  `json:"createdAt"`
}

type CheckDTO struct {
	ID        string    `json:"id"`
	CheckedAt time.Time `json:"checkedAt"`
	Up        bool      `json:"up"`
	LatencyMS int64     `json:"latencyMs"`
	Detail    string    `json:"detail,omitempty"`
}

type IncidentDTO struct {
	ID         string     `json:"id"`
	StartedAt  time.Time  `json:"startedAt"`
	ResolvedAt *time.Time `json:"resolvedAt"`
	Cause      string     `json:"cause"`
}

type UptimeSummaryDTO struct {
	Up    int      `json:"up"`
	Total int      `json:"total"`
	Ratio *float64 `json:"ratio"`
}

type MonitorUptimeDTO struct {
	Last24Hours UptimeSummaryDTO `json:"last24Hours"`
	Last7Days   UptimeSummaryDTO `json:"last7Days"`
	Last30Days  UptimeSummaryDTO `json:"last30Days"`
}

type PublicStatusDTO struct {
	Status   string                   `json:"status"`
	Monitors []PublicStatusMonitorDTO `json:"monitors"`
}

type PublicStatusMonitorDTO struct {
	Name          string           `json:"name"`
	Type          string           `json:"type"`
	Status        string           `json:"status"`
	LastCheckedAt *time.Time       `json:"lastCheckedAt"`
	Uptime        MonitorUptimeDTO `json:"uptime"`
	Checks        []PublicCheckDTO `json:"checks"`
}

type PublicCheckDTO struct {
	Up        bool  `json:"up"`
	LatencyMS int64 `json:"latencyMs"`
}

func userDTO(user domain.User, googleEnabled, telegramEnabled bool) UserDTO {
	return UserDTO{
		ID: string(user.ID), Name: user.Name, Email: user.Email,
		HasPassword: user.PasswordHash != "", GoogleLinked: user.GoogleSub != "",
		StatusSlug:   user.StatusSlug,
		Notify:       NotifyDTO{EmailAlerts: user.Notify.EmailAlerts, TelegramChatID: user.Notify.TelegramChatID},
		Capabilities: CapabilitiesDTO{Google: googleEnabled, Telegram: telegramEnabled},
	}
}

func monitorDTO(monitor domain.Monitor) MonitorDTO {
	return MonitorDTO{
		ID: string(monitor.ID), Name: monitor.Name, Type: string(monitor.Type), Target: monitor.Target,
		IntervalSeconds: monitor.Interval.Seconds(), TimeoutSeconds: monitor.Timeout.Seconds(),
		ExpectedStatus: monitor.ExpectedStatus, TLSWarnDays: monitor.TLSWarnDays,
		Public: monitor.Public, Enabled: monitor.Enabled, Status: string(monitor.Status),
		LastCheckedAt: optionalTime(monitor.LastCheckedAt), CreatedAt: monitor.CreatedAt,
	}
}

func checkDTO(check domain.Check) CheckDTO {
	return CheckDTO{
		ID: string(check.ID), CheckedAt: check.CheckedAt, Up: check.Up,
		LatencyMS: check.Latency.Milliseconds(), Detail: check.Detail,
	}
}

func incidentDTO(incident domain.Incident) IncidentDTO {
	return IncidentDTO{
		ID: string(incident.ID), StartedAt: incident.StartedAt,
		ResolvedAt: optionalTime(incident.ResolvedAt), Cause: incident.Cause,
	}
}

func monitorUptimeDTO(uptime service.MonitorUptime) MonitorUptimeDTO {
	return MonitorUptimeDTO{
		Last24Hours: uptimeSummaryDTO(uptime.Last24Hours),
		Last7Days:   uptimeSummaryDTO(uptime.Last7Days),
		Last30Days:  uptimeSummaryDTO(uptime.Last30Days),
	}
}

func publicStatusDTO(status service.PublicStatus) PublicStatusDTO {
	monitors := make([]PublicStatusMonitorDTO, 0, len(status.Monitors))
	for _, monitor := range status.Monitors {
		checks := make([]PublicCheckDTO, 0, len(monitor.Checks))
		for _, check := range monitor.Checks {
			checks = append(checks, PublicCheckDTO{Up: check.Up, LatencyMS: check.Latency.Milliseconds()})
		}
		monitors = append(monitors, PublicStatusMonitorDTO{
			Name: monitor.Name, Type: string(monitor.Type), Status: string(monitor.Status),
			LastCheckedAt: optionalTime(monitor.LastCheckedAt), Uptime: monitorUptimeDTO(monitor.Uptime), Checks: checks,
		})
	}
	return PublicStatusDTO{Status: string(status.Status), Monitors: monitors}
}

func uptimeSummaryDTO(summary service.UptimeSummary) UptimeSummaryDTO {
	return UptimeSummaryDTO{Up: summary.Up, Total: summary.Total, Ratio: summary.Ratio}
}

func optionalTime(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	return &value
}
