package service

import (
	"testing"

	"github.com/endale-dev/vigil/internal/domain"
)

func TestTransitionMonitor(t *testing.T) {
	for _, test := range []struct {
		name      string
		status    domain.Status
		failures  int
		probeUp   bool
		threshold int
		wantState domain.Status
		wantFails int
		changed   bool
	}{
		{name: "unknown to up", status: domain.StatusUnknown, probeUp: true, threshold: 2, wantState: domain.StatusUp, changed: true},
		{name: "down recovers", status: domain.StatusDown, failures: 4, probeUp: true, threshold: 2, wantState: domain.StatusUp, changed: true},
		{name: "up success unchanged", status: domain.StatusUp, failures: 1, probeUp: true, threshold: 2, wantState: domain.StatusUp},
		{name: "first failure keeps up", status: domain.StatusUp, failures: 0, threshold: 2, wantState: domain.StatusUp, wantFails: 1},
		{name: "threshold moves up down", status: domain.StatusUp, failures: 1, threshold: 2, wantState: domain.StatusDown, wantFails: 2, changed: true},
		{name: "unknown failure remains unknown", status: domain.StatusUnknown, failures: 0, threshold: 2, wantState: domain.StatusUnknown, wantFails: 1},
		{name: "threshold moves unknown down", status: domain.StatusUnknown, failures: 1, threshold: 2, wantState: domain.StatusDown, wantFails: 2, changed: true},
		{name: "down remains down", status: domain.StatusDown, failures: 2, threshold: 2, wantState: domain.StatusDown, wantFails: 3},
	} {
		t.Run(test.name, func(t *testing.T) {
			monitor := domain.Monitor{Status: test.status, ConsecutiveFailures: test.failures}
			status, failures := transitionMonitor(monitor, ProbeResult{Up: test.probeUp}, test.threshold)
			if status != test.wantState || failures != test.wantFails {
				t.Fatalf("transition = %s/%d, want %s/%d", status, failures, test.wantState, test.wantFails)
			}
			if changed := status != monitor.Status; changed != test.changed {
				t.Fatalf("changed = %v, want %v", changed, test.changed)
			}
		})
	}
}
