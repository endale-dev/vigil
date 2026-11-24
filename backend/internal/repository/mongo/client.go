package mongo

import (
	"context"
	"fmt"
	"time"

	driver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Client struct {
	client   *driver.Client
	database *driver.Database
}

func Connect(ctx context.Context, uri, database string) (*Client, error) {
	client, err := driver.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("create MongoDB client: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx, nil); err != nil {
		closeCtx, closeCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer closeCancel()
		_ = client.Disconnect(closeCtx)
		return nil, fmt.Errorf("ping MongoDB: %w", err)
	}

	return &Client{client: client, database: client.Database(database)}, nil
}

func (client *Client) Database() *driver.Database {
	return client.database
}

func (client *Client) Ping(ctx context.Context) error {
	return client.client.Ping(ctx, nil)
}

func (client *Client) Close(ctx context.Context) error {
	return client.client.Disconnect(ctx)
}
