// Created by HuyN on 2026/11/12.
// File middleware.go cung cấp HTTP middleware sử dụng TokenValidator để xác thực JWT
// từ header Authorization và đính kèm device_id vào request context.

package auth

import (
	"context"
	"net/http"
)

type contextKey string

const DeviceIDKey contextKey = "device_id"

// JWTMiddleware xác thực token thông qua TokenValidator và lưu device_id vào context
func JWTMiddleware(tv *TokenValidator, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
			return
		}

		// ParseAndValidate đã tự xử lý TrimSpace và TrimPrefix("Bearer ")
		claims, err := tv.ParseAndValidate(authHeader)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), DeviceIDKey, claims.DeviceID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetDeviceIDFromContext là hàm tiện ích giúp các handler lấy ra device_id đã xác thực
func GetDeviceIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(DeviceIDKey).(string)
	return id, ok
}
