//
// Created by HuyN on 2026/10/17.
// Description: Tầng truy xuất dữ liệu (Repository) cho collection telemetry; thực hiện ghi hàng loạt (InsertMany)
// và truy vấn lịch sử bản ghi cảm biến theo từng thiết bị.
//

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

// GetRecentByDevice lấy danh sách N bản ghi mới nhất của một thiết bị
func (r *TelemetryRepo) GetRecentByDevice(ctx context.Context, deviceID string, limit int64) ([]bson.M, error) {
	filter := bson.M{"device_id": deviceID}
	opts := options.Find().
		SetSort(bson.D{{Key: "timestamp", Value: -1}}). // Sắp xếp mới nhất lên đầu
		SetLimit(limit)

	cursor, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer func(cursor *mongo.Cursor, ctx context.Context) {
		err := cursor.Close(ctx)
		if err != nil {
			return
		}
	}(cursor, ctx)

	var results []bson.M
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	return results, nil
}
