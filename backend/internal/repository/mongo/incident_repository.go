package mongo

import (
	"context"
	"time"

	"github.com/endale-dev/vigil/internal/domain"
	apperrors "github.com/endale-dev/vigil/internal/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	driver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type incidentDocument struct {
	ID         bson.ObjectID `bson:"_id"`
	MonitorID  bson.ObjectID `bson:"monitorId"`
	StartedAt  time.Time     `bson:"startedAt"`
	ResolvedAt *time.Time    `bson:"resolvedAt,omitempty"`
	Cause      string        `bson:"cause"`
}

type incidentRepository struct {
	collection *driver.Collection
}

func NewIncidentRepository(database *driver.Database) domain.IncidentRepository {
	return &incidentRepository{collection: database.Collection("incidents")}
}

func (repository *incidentRepository) Open(ctx context.Context, incident *domain.Incident) error {
	if incident.ID == "" {
		incident.ID = objectIDToID(bson.NewObjectID())
	}
	document, err := incidentToDocument(incident)
	if err != nil {
		return err
	}
	if _, err := repository.collection.InsertOne(ctx, document); err != nil {
		return mapError(err)
	}
	return nil
}

func (repository *incidentRepository) Resolve(ctx context.Context, monitor domain.ID, at time.Time) error {
	monitorID, err := idToObjectID(monitor)
	if err != nil {
		return err
	}
	filter := bson.D{
		{Key: "monitorId", Value: monitorID},
		{Key: "$or", Value: bson.A{
			bson.D{{Key: "resolvedAt", Value: bson.D{{Key: "$exists", Value: false}}}},
			bson.D{{Key: "resolvedAt", Value: nil}},
		}},
	}
	result, err := repository.collection.UpdateOne(ctx, filter, bson.D{{Key: "$set", Value: bson.D{{Key: "resolvedAt", Value: at}}}})
	if err != nil {
		return mapError(err)
	}
	if result.MatchedCount == 0 {
		return apperrors.ErrNotFound
	}
	return nil
}

func (repository *incidentRepository) ListByMonitor(ctx context.Context, id domain.ID, limit int) ([]domain.Incident, error) {
	monitorID, err := idToObjectID(id)
	if err != nil {
		return nil, err
	}
	findOptions := options.Find().SetSort(bson.D{{Key: "startedAt", Value: -1}, {Key: "_id", Value: -1}})
	if limit > 0 {
		findOptions.SetLimit(int64(limit))
	}
	cursor, err := repository.collection.Find(ctx, bson.D{{Key: "monitorId", Value: monitorID}}, findOptions)
	if err != nil {
		return nil, mapError(err)
	}
	defer cursor.Close(ctx)
	var documents []incidentDocument
	if err := cursor.All(ctx, &documents); err != nil {
		return nil, mapError(err)
	}
	incidents := make([]domain.Incident, 0, len(documents))
	for i := range documents {
		incidents = append(incidents, *documents[i].toDomain())
	}
	return incidents, nil
}

func (repository *incidentRepository) DeleteByMonitor(ctx context.Context, id domain.ID) error {
	monitorID, err := idToObjectID(id)
	if err != nil {
		return err
	}
	_, err = repository.collection.DeleteMany(ctx, bson.D{{Key: "monitorId", Value: monitorID}})
	return mapError(err)
}

func incidentToDocument(incident *domain.Incident) (*incidentDocument, error) {
	id, err := idToObjectID(incident.ID)
	if err != nil {
		return nil, err
	}
	monitorID, err := idToObjectID(incident.MonitorID)
	if err != nil {
		return nil, err
	}
	document := &incidentDocument{
		ID:        id,
		MonitorID: monitorID,
		StartedAt: incident.StartedAt,
		Cause:     incident.Cause,
	}
	if !incident.ResolvedAt.IsZero() {
		resolvedAt := incident.ResolvedAt
		document.ResolvedAt = &resolvedAt
	}
	return document, nil
}

func (document *incidentDocument) toDomain() *domain.Incident {
	incident := &domain.Incident{
		ID:        objectIDToID(document.ID),
		MonitorID: objectIDToID(document.MonitorID),
		StartedAt: document.StartedAt,
		Cause:     document.Cause,
	}
	if document.ResolvedAt != nil {
		incident.ResolvedAt = *document.ResolvedAt
	}
	return incident
}

var _ domain.IncidentRepository = (*incidentRepository)(nil)
