package mongo

import (
	"testing"
	"time"

	"github.com/endale-dev/vigil/internal/domain"
)

func TestMonitorDocumentMapping(t *testing.T) {
	want := &domain.Monitor{
		ID:                  "507f1f77bcf86cd799439011",
		OwnerID:             "507f1f77bcf86cd799439012",
		Name:                "API",
		Type:                domain.MonitorHTTP,
		Target:              "https://example.com/health",
		Interval:            time.Minute,
		Timeout:             10 * time.Second,
		ExpectedStatus:      200,
		Public:              true,
		Enabled:             true,
		Status:              domain.StatusUp,
		ConsecutiveFailures: 0,
		LastCheckedAt:       time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC),
		NextCheckAt:         time.Date(2026, time.January, 2, 3, 5, 5, 0, time.UTC),
		CreatedAt:           time.Date(2026, time.January, 1, 3, 4, 5, 0, time.UTC),
	}
	document, err := monitorToDocument(want)
	if err != nil {
		t.Fatal(err)
	}
	got := document.toDomain()
	if *got != *want {
		t.Fatalf("monitor mapping mismatch: got %+v, want %+v", got, want)
	}
}
