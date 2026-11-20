package mongo

import (
	"testing"
	"time"

	"github.com/endale-dev/vigil/internal/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestIncidentDocumentRoundTrip(t *testing.T) {
	id := bson.NewObjectID()
	monitorID := bson.NewObjectID()
	startedAt := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	resolvedAt := startedAt.Add(10 * time.Minute)
	want := &domain.Incident{
		ID:         objectIDToID(id),
		MonitorID:  objectIDToID(monitorID),
		StartedAt:  startedAt,
		ResolvedAt: resolvedAt,
		Cause:      "HTTP returned 503",
	}

	document, err := incidentToDocument(want)
	if err != nil {
		t.Fatalf("incidentToDocument() error = %v", err)
	}
	got := document.toDomain()
	if *got != *want {
		t.Fatalf("round trip = %#v, want %#v", got, want)
	}
}

func TestOpenIncidentOmitsResolvedAt(t *testing.T) {
	document, err := incidentToDocument(&domain.Incident{
		ID:        domain.ID(bson.NewObjectID().Hex()),
		MonitorID: domain.ID(bson.NewObjectID().Hex()),
	})
	if err != nil {
		t.Fatalf("incidentToDocument() error = %v", err)
	}
	encoded, err := bson.Marshal(document)
	if err != nil {
		t.Fatalf("bson.Marshal() error = %v", err)
	}
	var fields bson.M
	if err := bson.Unmarshal(encoded, &fields); err != nil {
		t.Fatalf("bson.Unmarshal() error = %v", err)
	}
	if _, ok := fields["resolvedAt"]; ok {
		t.Fatalf("open incident encoded resolvedAt: %#v", fields["resolvedAt"])
	}
}
