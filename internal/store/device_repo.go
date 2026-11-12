//
// Created by HuyN on 2026/10/17.
// Description: Quản lý thông tin định danh các nút cảm biến (IoT nodes) trong MongoDB và sinh mã JWT token
// thời hạn dài để nạp cho vi điều khiển.
//

package store

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type Device struct {
	ID        string    `bson:"_id" json:"device_id"`
	Name      string    `bson:"name" json:"name"`
	SecretKey string    `bson:"secret_key" json:"-"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
}

type DeviceRepo struct {
	coll      *mongo.Collection
	jwtSecret []byte
}

func NewDeviceRepo(m *DB, jwtSecret []byte) *DeviceRepo {
	return &DeviceRepo{
		coll:      m.client.Database(m.Name).Collection("devices"),
		jwtSecret: jwtSecret,
	}
}

// RegisterDevice lưu device vào MongoDB và cấp một JWT token có hạn 1 năm
func (r *DeviceRepo) RegisterDevice(ctx context.Context, deviceID, name string) (string, error) {
	if deviceID == "" {
		return "", errors.New("device_id không được để trống")
	}

	dev := Device{
		ID:        deviceID,
		Name:      name,
		CreatedAt: time.Now(),
	}

	// Lưu hoặc cập nhật thông tin thiết bị
	_, err := r.coll.InsertOne(ctx, dev)
	if err != nil && !mongo.IsDuplicateKeyError(err) {
		return "", err
	}

	// Sinh JWT token có hạn 365 ngày cho thiết bị nhúng
	claims := jwt.MapClaims{
		"device_id": deviceID,
		"exp":       time.Now().Add(365 * 24 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(r.jwtSecret)
	if err != nil {
		return "", err
	}

	return tokenStr, nil
}
