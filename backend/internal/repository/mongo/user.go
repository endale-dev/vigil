package mongo

import (
	"context"
	"time"

	"github.com/endale-dev/vigil/internal/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	driver "go.mongodb.org/mongo-driver/v2/mongo"
)

type userDocument struct {
	ID           bson.ObjectID  `bson:"_id"`
	Name         string         `bson:"name"`
	Email        string         `bson:"email"`
	PasswordHash string         `bson:"passwordHash"`
	GoogleSub    string         `bson:"googleSub"`
	StatusSlug   string         `bson:"statusSlug"`
	Notify       notifyDocument `bson:"notify"`
	CreatedAt    time.Time      `bson:"createdAt"`
}

type notifyDocument struct {
	EmailAlerts    bool   `bson:"emailAlerts"`
	TelegramChatID string `bson:"telegramChatId"`
}

type userRepository struct {
	collection *driver.Collection
}

func NewUserRepository(database *driver.Database) domain.UserRepository {
	return &userRepository{collection: database.Collection("users")}
}

func (repository *userRepository) Create(ctx context.Context, user *domain.User) error {
	if user.ID == "" {
		user.ID = objectIDToID(bson.NewObjectID())
	}
	document, err := userToDocument(user)
	if err != nil {
		return err
	}
	if _, err := repository.collection.InsertOne(ctx, document); err != nil {
		return mapError(err)
	}
	return nil
}

func (repository *userRepository) GetByID(ctx context.Context, id domain.ID) (*domain.User, error) {
	objectID, err := idToObjectID(id)
	if err != nil {
		return nil, err
	}
	return repository.find(ctx, bson.D{{Key: "_id", Value: objectID}})
}

func (repository *userRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return repository.find(ctx, bson.D{{Key: "email", Value: email}})
}

func (repository *userRepository) GetByGoogleSub(ctx context.Context, sub string) (*domain.User, error) {
	return repository.find(ctx, bson.D{{Key: "googleSub", Value: sub}})
}

func (repository *userRepository) GetBySlug(ctx context.Context, slug string) (*domain.User, error) {
	return repository.find(ctx, bson.D{{Key: "statusSlug", Value: slug}})
}

func (repository *userRepository) Update(ctx context.Context, user *domain.User) error {
	objectID, err := idToObjectID(user.ID)
	if err != nil {
		return err
	}
	document, err := userToDocument(user)
	if err != nil {
		return err
	}
	result, err := repository.collection.ReplaceOne(ctx, bson.D{{Key: "_id", Value: objectID}}, document)
	if err != nil {
		return mapError(err)
	}
	if result.MatchedCount == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (repository *userRepository) find(ctx context.Context, filter bson.D) (*domain.User, error) {
	var document userDocument
	if err := repository.collection.FindOne(ctx, filter).Decode(&document); err != nil {
		return nil, mapError(err)
	}
	return document.toDomain(), nil
}

func userToDocument(user *domain.User) (*userDocument, error) {
	objectID, err := idToObjectID(user.ID)
	if err != nil {
		return nil, err
	}
	return &userDocument{
		ID:           objectID,
		Name:         user.Name,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		GoogleSub:    user.GoogleSub,
		StatusSlug:   user.StatusSlug,
		Notify: notifyDocument{
			EmailAlerts:    user.Notify.EmailAlerts,
			TelegramChatID: user.Notify.TelegramChatID,
		},
		CreatedAt: user.CreatedAt,
	}, nil
}

func (document *userDocument) toDomain() *domain.User {
	return &domain.User{
		ID:           objectIDToID(document.ID),
		Name:         document.Name,
		Email:        document.Email,
		PasswordHash: document.PasswordHash,
		GoogleSub:    document.GoogleSub,
		StatusSlug:   document.StatusSlug,
		Notify: domain.NotifySettings{
			EmailAlerts:    document.Notify.EmailAlerts,
			TelegramChatID: document.Notify.TelegramChatID,
		},
		CreatedAt: document.CreatedAt,
	}
}

var _ domain.UserRepository = (*userRepository)(nil)
