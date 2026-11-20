package service

import "github.com/endale-dev/vigil/internal/domain"

func transitionMonitor(monitor domain.Monitor, result ProbeResult, failureThreshold int) (domain.Status, int) {
	if result.Up {
		return domain.StatusUp, 0
	}
	failures := monitor.ConsecutiveFailures + 1
	if failures >= failureThreshold {
		return domain.StatusDown, failures
	}
	return monitor.Status, failures
}
