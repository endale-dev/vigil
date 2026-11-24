package mongo

import (
	"context"
	"fmt"
	"math"
	"time"

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

func CreateMonitorIndexes(ctx context.Context, database *driver.Database) error {
	models := []driver.IndexModel{
		{
			Keys:    bson.D{{Key: "ownerId", Value: 1}},
			Options: options.Index().SetName("monitors_owner"),
		},
		{
			Keys: bson.D{
				{Key: "enabled", Value: 1},
				{Key: "nextCheckAt", Value: 1},
				{Key: "leaseUntil", Value: 1},
				{Key: "_id", Value: 1},
			},
			Options: options.Index().SetName("monitors_due_lease"),
		},
	}
	if _, err := database.Collection("monitors").Indexes().CreateMany(ctx, models); err != nil {
		return fmt.Errorf("create monitor indexes: %w", err)
	}
	return nil
}

func CreateCheckIndexes(ctx context.Context, database *driver.Database, retention time.Duration) error {
	seconds := int64(retention / time.Second)
	if retention%time.Second != 0 {
		seconds++
	}
	if seconds < 1 || seconds > math.MaxInt32 {
		return fmt.Errorf("check retention must be between 1 second and %d seconds", math.MaxInt32)
	}
	models := []driver.IndexModel{
		{
			Keys:    bson.D{{Key: "monitorId", Value: 1}, {Key: "checkedAt", Value: -1}},
			Options: options.Index().SetName("checks_monitor_checked_at"),
		},
		{
			Keys:    bson.D{{Key: "checkedAt", Value: 1}},
			Options: options.Index().SetName("checks_checked_at_ttl").SetExpireAfterSeconds(int32(seconds)),
		},
	}
	if _, err := database.Collection("checks").Indexes().CreateMany(ctx, models); err != nil {
		return fmt.Errorf("create check indexes: %w", err)
	}
	return nil
}

func CreateIncidentIndexes(ctx context.Context, database *driver.Database) error {
	models := []driver.IndexModel{
		{
			Keys:    bson.D{{Key: "monitorId", Value: 1}, {Key: "startedAt", Value: -1}},
			Options: options.Index().SetName("incidents_monitor_started_at"),
		},
		{
			Keys: bson.D{{Key: "monitorId", Value: 1}},
			Options: options.Index().SetName("incidents_one_open_per_monitor").SetUnique(true).
				SetPartialFilterExpression(bson.D{{Key: "resolvedAt", Value: nil}}),
		},
	}
	if _, err := database.Collection("incidents").Indexes().CreateMany(ctx, models); err != nil {
		return fmt.Errorf("create incident indexes: %w", err)
	}
	return nil
}

func CreateIndexes(ctx context.Context, database *driver.Database, retention time.Duration) error {
	for _, create := range []func(context.Context, *driver.Database) error{
		CreateUserIndexes,
		CreateSessionIndexes,
		CreatePasswordResetIndexes,
		CreateMonitorIndexes,
		CreateIncidentIndexes,
	} {
		if err := create(ctx, database); err != nil {
			return err
		}
	}
	return CreateCheckIndexes(ctx, database, retention)
}
