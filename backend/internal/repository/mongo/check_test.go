package mongo

import (
	"testing"
	"time"

	"github.com/endale-dev/vigil/internal/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestCheckDocumentRoundTrip(t *testing.T) {
	id := bson.NewObjectID()
	monitorID := bson.NewObjectID()
	checkedAt := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	want := &domain.Check{
		ID:        objectIDToID(id),
		MonitorID: objectIDToID(monitorID),
		CheckedAt: checkedAt,
		Up:        true,
		Latency:   1234 * time.Millisecond,
		Detail:    "HTTP 200",
	}

	document, err := checkToDocument(want)
	if err != nil {
		t.Fatalf("checkToDocument() error = %v", err)
	}
	got := document.toDomain()
	if *got != *want {
		t.Fatalf("round trip = %#v, want %#v", got, want)
	}
}

func TestCheckToDocumentRejectsInvalidMonitorID(t *testing.T) {
	_, err := checkToDocument(&domain.Check{ID: domain.ID(bson.NewObjectID().Hex()), MonitorID: "invalid"})
	if err != domain.ErrNotFound {
		t.Fatalf("checkToDocument() error = %v, want %v", err, domain.ErrNotFound)
	}
}
