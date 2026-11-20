package mongo

import (
	"context"
	"time"

	"github.com/endale-dev/vigil/internal/domain"
	apperrors "github.com/endale-dev/vigil/internal/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	driver "go.mongodb.org/mongo-driver/v2/mongo"
)

type sessionDocument struct {
	ID        bson.ObjectID `bson:"_id"`
	UserID    bson.ObjectID `bson:"userId"`
	TokenHash string        `bson:"tokenHash"`
	ExpiresAt time.Time     `bson:"expiresAt"`
	CreatedAt time.Time     `bson:"createdAt"`
}

type sessionRepository struct {
	collection *driver.Collection
}

func NewSessionRepository(database *driver.Database) domain.SessionRepository {
	return &sessionRepository{collection: database.Collection("sessions")}
}

func (repository *sessionRepository) Create(ctx context.Context, session *domain.Session) error {
	if session.ID == "" {
		session.ID = objectIDToID(bson.NewObjectID())
	}
	document, err := sessionToDocument(session)
	if err != nil {
		return err
	}
	if _, err := repository.collection.InsertOne(ctx, document); err != nil {
		return mapError(err)
	}
	return nil
}

func (repository *sessionRepository) GetByTokenHash(ctx context.Context, hash string) (*domain.Session, error) {
	var document sessionDocument
	if err := repository.collection.FindOne(ctx, bson.D{{Key: "tokenHash", Value: hash}}).Decode(&document); err != nil {
		return nil, mapError(err)
	}
	return document.toDomain(), nil
}

func (repository *sessionRepository) Delete(ctx context.Context, id domain.ID) error {
	objectID, err := idToObjectID(id)
	if err != nil {
		return err
	}
	result, err := repository.collection.DeleteOne(ctx, bson.D{{Key: "_id", Value: objectID}})
	if err != nil {
		return mapError(err)
	}
	if result.DeletedCount == 0 {
		return apperrors.ErrNotFound
	}
	return nil
}

func (repository *sessionRepository) DeleteByUserExcept(ctx context.Context, userID, keep domain.ID) error {
	userObjectID, err := idToObjectID(userID)
	if err != nil {
		return err
	}
	keepObjectID, err := idToObjectID(keep)
	if err != nil {
		return err
	}
	_, err = repository.collection.DeleteMany(ctx, bson.D{
		{Key: "userId", Value: userObjectID},
		{Key: "_id", Value: bson.D{{Key: "$ne", Value: keepObjectID}}},
	})
	return mapError(err)
}

func (repository *sessionRepository) DeleteByUser(ctx context.Context, userID domain.ID) error {
	userObjectID, err := idToObjectID(userID)
	if err != nil {
		return err
	}
	_, err = repository.collection.DeleteMany(ctx, bson.D{{Key: "userId", Value: userObjectID}})
	return mapError(err)
}

func sessionToDocument(session *domain.Session) (*sessionDocument, error) {
	id, err := idToObjectID(session.ID)
	if err != nil {
		return nil, err
	}
	userID, err := idToObjectID(session.UserID)
	if err != nil {
		return nil, err
	}
	return &sessionDocument{
		ID:        id,
		UserID:    userID,
		TokenHash: session.TokenHash,
		ExpiresAt: session.ExpiresAt,
		CreatedAt: session.CreatedAt,
	}, nil
}

func (document *sessionDocument) toDomain() *domain.Session {
	return &domain.Session{
		ID:        objectIDToID(document.ID),
		UserID:    objectIDToID(document.UserID),
		TokenHash: document.TokenHash,
		ExpiresAt: document.ExpiresAt,
		CreatedAt: document.CreatedAt,
	}
}

var _ domain.SessionRepository = (*sessionRepository)(nil)
