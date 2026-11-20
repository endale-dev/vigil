package mongo

import (
	"context"
	"time"

	"github.com/endale-dev/vigil/internal/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	driver "go.mongodb.org/mongo-driver/v2/mongo"
)

type passwordResetDocument struct {
	ID        bson.ObjectID `bson:"_id"`
	UserID    bson.ObjectID `bson:"userId"`
	TokenHash string        `bson:"tokenHash"`
	ExpiresAt time.Time     `bson:"expiresAt"`
}

type passwordResetRepository struct {
	collection *driver.Collection
}

func NewPasswordResetRepository(database *driver.Database) domain.PasswordResetRepository {
	return &passwordResetRepository{collection: database.Collection("password_resets")}
}

func (repository *passwordResetRepository) Create(ctx context.Context, reset *domain.PasswordReset) error {
	if reset.ID == "" {
		reset.ID = objectIDToID(bson.NewObjectID())
	}
	document, err := passwordResetToDocument(reset)
	if err != nil {
		return err
	}
	if _, err := repository.collection.InsertOne(ctx, document); err != nil {
		return mapError(err)
	}
	return nil
}

func (repository *passwordResetRepository) GetByTokenHash(ctx context.Context, hash string) (*domain.PasswordReset, error) {
	var document passwordResetDocument
	if err := repository.collection.FindOne(ctx, bson.D{{Key: "tokenHash", Value: hash}}).Decode(&document); err != nil {
		return nil, mapError(err)
	}
	return document.toDomain(), nil
}

func (repository *passwordResetRepository) DeleteByUser(ctx context.Context, userID domain.ID) error {
	objectID, err := idToObjectID(userID)
	if err != nil {
		return err
	}
	_, err = repository.collection.DeleteMany(ctx, bson.D{{Key: "userId", Value: objectID}})
	return mapError(err)
}

func passwordResetToDocument(reset *domain.PasswordReset) (*passwordResetDocument, error) {
	id, err := idToObjectID(reset.ID)
	if err != nil {
		return nil, err
	}
	userID, err := idToObjectID(reset.UserID)
	if err != nil {
		return nil, err
	}
	return &passwordResetDocument{
		ID:        id,
		UserID:    userID,
		TokenHash: reset.TokenHash,
		ExpiresAt: reset.ExpiresAt,
	}, nil
}

func (document *passwordResetDocument) toDomain() *domain.PasswordReset {
	return &domain.PasswordReset{
		ID:        objectIDToID(document.ID),
		UserID:    objectIDToID(document.UserID),
		TokenHash: document.TokenHash,
		ExpiresAt: document.ExpiresAt,
	}
}

var _ domain.PasswordResetRepository = (*passwordResetRepository)(nil)
