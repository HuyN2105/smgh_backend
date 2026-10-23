package store

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type DB struct {
	client *mongo.Client
	Name   string
}

func NewMongoDB(ctx context.Context, uri, dbName string) (*DB, error) {
	clientOpts := options.Client().
		ApplyURI(uri).
		SetMaxPoolSize(100).
		SetMinPoolSize(10).
		SetMaxConnIdleTime(60 * time.Second)
	client, err := mongo.Connect(clientOpts)
	if err != nil {
		return nil, err
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	return &DB{client, dbName}, nil
}

func (d *DB) Close(ctx context.Context) error {
	return d.client.Disconnect(ctx)
}
