package store

import (
	"CentralServer/internal/telemetry"
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const DateTimeLayout = "2006/01/02 15:04:05"

type TelemetryDoc struct {
	DeviceID  string             `bson:"device_id"`
	Timestamp string             `bson:"timestamp"`
	Metrics   map[string]float64 `bson:"metrics"`
	CreatedAt string             `bson:"created_at"`
}

type TelemetryRepo struct {
	coll *mongo.Collection
}

func NewTelemetryRepo(ctx context.Context, db *DB) (*TelemetryRepo, error) {
	coll := db.client.Database(db.Name).Collection("sensor_telemetry")

	// Index: (device_id ASC, ts DESC) for time-series range scans
	indexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "device_id", Value: 1},
			{Key: "timestamp", Value: -1},
		},
	}

	_, err := coll.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		return nil, err
	}

	return &TelemetryRepo{coll: coll}, nil
}

func (r *TelemetryRepo) InsertBatch(ctx context.Context, readings []telemetry.SensorReading) error {
	if len(readings) == 0 {
		return nil
	}

	docs := make([]any, len(readings))
	nowStr := time.Now().Format(DateTimeLayout)

	for i, rd := range readings {
		ts := rd.Timestamp
		if ts == "" {
			ts = nowStr
		}

		docs[i] = TelemetryDoc{
			DeviceID:  rd.DeviceID,
			Timestamp: ts,
			Metrics:   rd.Metrics,
			CreatedAt: nowStr,
		}
	}

	_, err := r.coll.InsertMany(ctx, docs, options.InsertMany().SetOrdered(false))
	return err
}
