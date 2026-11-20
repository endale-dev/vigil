package mongo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	driver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func CreateUserIndexes(ctx context.Context, database *driver.Database) error {
	models := []driver.IndexModel{
		{
			Keys:    bson.D{{Key: "email", Value: 1}},
			Options: options.Index().SetName("users_email_unique").SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "statusSlug", Value: 1}},
			Options: options.Index().SetName("users_status_slug_unique").SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "googleSub", Value: 1}},
			Options: options.Index().SetName("users_google_sub_unique").SetUnique(true).
				SetPartialFilterExpression(bson.D{{Key: "googleSub", Value: bson.D{{Key: "$gt", Value: ""}}}}),
		},
	}
	if _, err := database.Collection("users").Indexes().CreateMany(ctx, models); err != nil {
		return fmt.Errorf("create user indexes: %w", err)
	}
	return nil
}

func CreateSessionIndexes(ctx context.Context, database *driver.Database) error {
	models := []driver.IndexModel{
		{
			Keys:    bson.D{{Key: "tokenHash", Value: 1}},
			Options: options.Index().SetName("sessions_token_hash_unique").SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "expiresAt", Value: 1}},
			Options: options.Index().SetName("sessions_expires_at_ttl").SetExpireAfterSeconds(0),
		},
	}
	if _, err := database.Collection("sessions").Indexes().CreateMany(ctx, models); err != nil {
		return fmt.Errorf("create session indexes: %w", err)
	}
	return nil
}

func CreatePasswordResetIndexes(ctx context.Context, database *driver.Database) error {
	models := []driver.IndexModel{
		{
			Keys:    bson.D{{Key: "tokenHash", Value: 1}},
			Options: options.Index().SetName("password_resets_token_hash_unique").SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "expiresAt", Value: 1}},
			Options: options.Index().SetName("password_resets_expires_at_ttl").SetExpireAfterSeconds(0),
		},
	}
	if _, err := database.Collection("password_resets").Indexes().CreateMany(ctx, models); err != nil {
		return fmt.Errorf("create password reset indexes: %w", err)
	}
	return nil
}
