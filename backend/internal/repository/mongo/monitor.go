package mongo

import (
	"context"
	"time"

	"github.com/endale-dev/vigil/internal/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	driver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type monitorDocument struct {
	ID                  bson.ObjectID      `bson:"_id"`
	OwnerID             bson.ObjectID      `bson:"ownerId"`
	Name                string             `bson:"name"`
	Type                domain.MonitorType `bson:"type"`
	Target              string             `bson:"target"`
	IntervalSeconds     int64              `bson:"intervalSeconds"`
	TimeoutSeconds      int64              `bson:"timeoutSeconds"`
	ExpectedStatus      int                `bson:"expectedStatus"`
	TLSWarnDays         int                `bson:"tlsWarnDays"`
	Public              bool               `bson:"public"`
	Enabled             bool               `bson:"enabled"`
	Status              domain.Status      `bson:"status"`
	ConsecutiveFailures int                `bson:"consecutiveFailures"`
	LastCheckedAt       time.Time          `bson:"lastCheckedAt"`
	NextCheckAt         time.Time          `bson:"nextCheckAt"`
	LeaseUntil          *time.Time         `bson:"leaseUntil,omitempty"`
	ClaimToken          string             `bson:"claimToken,omitempty"`
	CreatedAt           time.Time          `bson:"createdAt"`
}

type monitorRepository struct {
	collection *driver.Collection
}

func NewMonitorRepository(database *driver.Database) *monitorRepository {
	return &monitorRepository{collection: database.Collection("monitors")}
}

func (repository *monitorRepository) Create(ctx context.Context, monitor *domain.Monitor) error {
	if monitor.ID == "" {
		monitor.ID = objectIDToID(bson.NewObjectID())
	}
	document, err := monitorToDocument(monitor)
	if err != nil {
		return err
	}
	if _, err := repository.collection.InsertOne(ctx, document); err != nil {
		return mapError(err)
	}
	return nil
}

func (repository *monitorRepository) GetByID(ctx context.Context, id domain.ID) (*domain.Monitor, error) {
	objectID, err := idToObjectID(id)
	if err != nil {
		return nil, err
	}
	var document monitorDocument
	if err := repository.collection.FindOne(ctx, bson.D{{Key: "_id", Value: objectID}}).Decode(&document); err != nil {
		return nil, mapError(err)
	}
	return document.toDomain(), nil
}

func (repository *monitorRepository) ListByOwner(ctx context.Context, owner domain.ID) ([]domain.Monitor, error) {
	ownerID, err := idToObjectID(owner)
	if err != nil {
		return nil, err
	}
	return repository.list(ctx, bson.D{{Key: "ownerId", Value: ownerID}})
}

func (repository *monitorRepository) ListPublicByOwner(ctx context.Context, owner domain.ID) ([]domain.Monitor, error) {
	ownerID, err := idToObjectID(owner)
	if err != nil {
		return nil, err
	}
	return repository.list(ctx, bson.D{
		{Key: "ownerId", Value: ownerID},
		{Key: "public", Value: true},
	})
}

func (repository *monitorRepository) CountByOwner(ctx context.Context, owner domain.ID) (int, error) {
	ownerID, err := idToObjectID(owner)
	if err != nil {
		return 0, err
	}
	count, err := repository.collection.CountDocuments(ctx, bson.D{{Key: "ownerId", Value: ownerID}})
	if err != nil {
		return 0, mapError(err)
	}
	return int(count), nil
}

func (repository *monitorRepository) Update(ctx context.Context, monitor *domain.Monitor) error {
	document, err := monitorToDocument(monitor)
	if err != nil {
		return err
	}
	set := bson.D{
		{Key: "ownerId", Value: document.OwnerID},
		{Key: "name", Value: document.Name},
		{Key: "type", Value: document.Type},
		{Key: "target", Value: document.Target},
		{Key: "intervalSeconds", Value: document.IntervalSeconds},
		{Key: "timeoutSeconds", Value: document.TimeoutSeconds},
		{Key: "expectedStatus", Value: document.ExpectedStatus},
		{Key: "tlsWarnDays", Value: document.TLSWarnDays},
		{Key: "public", Value: document.Public},
		{Key: "enabled", Value: document.Enabled},
		{Key: "status", Value: document.Status},
		{Key: "consecutiveFailures", Value: document.ConsecutiveFailures},
		{Key: "lastCheckedAt", Value: document.LastCheckedAt},
		{Key: "nextCheckAt", Value: document.NextCheckAt},
	}
	result, err := repository.collection.UpdateOne(ctx, bson.D{{Key: "_id", Value: document.ID}}, bson.D{{Key: "$set", Value: set}})
	if err != nil {
		return mapError(err)
	}
	if result.MatchedCount == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (repository *monitorRepository) Delete(ctx context.Context, id domain.ID) error {
	objectID, err := idToObjectID(id)
	if err != nil {
		return err
	}
	result, err := repository.collection.DeleteOne(ctx, bson.D{{Key: "_id", Value: objectID}})
	if err != nil {
		return mapError(err)
	}
	if result.DeletedCount == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (repository *monitorRepository) list(ctx context.Context, filter bson.D) ([]domain.Monitor, error) {
	cursor, err := repository.collection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}))
	if err != nil {
		return nil, mapError(err)
	}
	defer cursor.Close(ctx)
	var documents []monitorDocument
	if err := cursor.All(ctx, &documents); err != nil {
		return nil, mapError(err)
	}
	monitors := make([]domain.Monitor, 0, len(documents))
	for i := range documents {
		monitors = append(monitors, *documents[i].toDomain())
	}
	return monitors, nil
}

func monitorToDocument(monitor *domain.Monitor) (*monitorDocument, error) {
	id, err := idToObjectID(monitor.ID)
	if err != nil {
		return nil, err
	}
	ownerID, err := idToObjectID(monitor.OwnerID)
	if err != nil {
		return nil, err
	}
	return &monitorDocument{
		ID:                  id,
		OwnerID:             ownerID,
		Name:                monitor.Name,
		Type:                monitor.Type,
		Target:              monitor.Target,
		IntervalSeconds:     int64(monitor.Interval / time.Second),
		TimeoutSeconds:      int64(monitor.Timeout / time.Second),
		ExpectedStatus:      monitor.ExpectedStatus,
		TLSWarnDays:         monitor.TLSWarnDays,
		Public:              monitor.Public,
		Enabled:             monitor.Enabled,
		Status:              monitor.Status,
		ConsecutiveFailures: monitor.ConsecutiveFailures,
		LastCheckedAt:       monitor.LastCheckedAt,
		NextCheckAt:         monitor.NextCheckAt,
		CreatedAt:           monitor.CreatedAt,
	}, nil
}

func (document *monitorDocument) toDomain() *domain.Monitor {
	return &domain.Monitor{
		ID:                  objectIDToID(document.ID),
		OwnerID:             objectIDToID(document.OwnerID),
		Name:                document.Name,
		Type:                document.Type,
		Target:              document.Target,
		Interval:            time.Duration(document.IntervalSeconds) * time.Second,
		Timeout:             time.Duration(document.TimeoutSeconds) * time.Second,
		ExpectedStatus:      document.ExpectedStatus,
		TLSWarnDays:         document.TLSWarnDays,
		Public:              document.Public,
		Enabled:             document.Enabled,
		Status:              document.Status,
		ConsecutiveFailures: document.ConsecutiveFailures,
		LastCheckedAt:       document.LastCheckedAt,
		NextCheckAt:         document.NextCheckAt,
		CreatedAt:           document.CreatedAt,
	}
}
