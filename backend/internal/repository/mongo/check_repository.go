package mongo

import (
	"context"
	"time"

	"github.com/endale-dev/vigil/internal/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	driver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type checkDocument struct {
	ID        bson.ObjectID `bson:"_id"`
	MonitorID bson.ObjectID `bson:"monitorId"`
	CheckedAt time.Time     `bson:"checkedAt"`
	Up        bool          `bson:"up"`
	LatencyMS int64         `bson:"latencyMs"`
	Detail    string        `bson:"detail"`
}

type checkRepository struct {
	collection *driver.Collection
}

func NewCheckRepository(database *driver.Database) domain.CheckRepository {
	return &checkRepository{collection: database.Collection("checks")}
}

func (repository *checkRepository) Insert(ctx context.Context, check *domain.Check) error {
	if check.ID == "" {
		check.ID = objectIDToID(bson.NewObjectID())
	}
	document, err := checkToDocument(check)
	if err != nil {
		return err
	}
	if _, err := repository.collection.InsertOne(ctx, document); err != nil {
		return mapError(err)
	}
	return nil
}

func (repository *checkRepository) ListByMonitor(ctx context.Context, id domain.ID, limit int) ([]domain.Check, error) {
	monitorID, err := idToObjectID(id)
	if err != nil {
		return nil, err
	}
	findOptions := options.Find().SetSort(bson.D{{Key: "checkedAt", Value: -1}, {Key: "_id", Value: -1}})
	if limit > 0 {
		findOptions.SetLimit(int64(limit))
	}
	cursor, err := repository.collection.Find(ctx, bson.D{{Key: "monitorId", Value: monitorID}}, findOptions)
	if err != nil {
		return nil, mapError(err)
	}
	defer cursor.Close(ctx)
	var documents []checkDocument
	if err := cursor.All(ctx, &documents); err != nil {
		return nil, mapError(err)
	}
	checks := make([]domain.Check, 0, len(documents))
	for i := range documents {
		checks = append(checks, *documents[i].toDomain())
	}
	return checks, nil
}

func (repository *checkRepository) UptimeSince(ctx context.Context, id domain.ID, since time.Time) (domain.Uptime, error) {
	monitorID, err := idToObjectID(id)
	if err != nil {
		return domain.Uptime{}, err
	}
	pipeline := driver.Pipeline{
		bson.D{{Key: "$match", Value: bson.D{
			{Key: "monitorId", Value: monitorID},
			{Key: "checkedAt", Value: bson.D{{Key: "$gte", Value: since}}},
		}}},
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: nil},
			{Key: "up", Value: bson.D{{Key: "$sum", Value: bson.D{{Key: "$cond", Value: bson.A{"$up", 1, 0}}}}}},
			{Key: "total", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
	}
	cursor, err := repository.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return domain.Uptime{}, mapError(err)
	}
	defer cursor.Close(ctx)
	var result []struct {
		Up    int `bson:"up"`
		Total int `bson:"total"`
	}
	if err := cursor.All(ctx, &result); err != nil {
		return domain.Uptime{}, mapError(err)
	}
	if len(result) == 0 {
		return domain.Uptime{}, nil
	}
	return domain.Uptime{Up: result[0].Up, Total: result[0].Total}, nil
}

func (repository *checkRepository) DeleteByMonitor(ctx context.Context, id domain.ID) error {
	monitorID, err := idToObjectID(id)
	if err != nil {
		return err
	}
	_, err = repository.collection.DeleteMany(ctx, bson.D{{Key: "monitorId", Value: monitorID}})
	return mapError(err)
}

func checkToDocument(check *domain.Check) (*checkDocument, error) {
	id, err := idToObjectID(check.ID)
	if err != nil {
		return nil, err
	}
	monitorID, err := idToObjectID(check.MonitorID)
	if err != nil {
		return nil, err
	}
	return &checkDocument{
		ID:        id,
		MonitorID: monitorID,
		CheckedAt: check.CheckedAt,
		Up:        check.Up,
		LatencyMS: int64(check.Latency / time.Millisecond),
		Detail:    check.Detail,
	}, nil
}

func (document *checkDocument) toDomain() *domain.Check {
	return &domain.Check{
		ID:        objectIDToID(document.ID),
		MonitorID: objectIDToID(document.MonitorID),
		CheckedAt: document.CheckedAt,
		Up:        document.Up,
		Latency:   time.Duration(document.LatencyMS) * time.Millisecond,
		Detail:    document.Detail,
	}
}

var _ domain.CheckRepository = (*checkRepository)(nil)
