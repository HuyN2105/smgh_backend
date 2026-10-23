package auth

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type DeviceClaims struct {
	DeviceID string `json:"device_id"`
	jwt.RegisteredClaims
}

type TokenValidator struct {
	secret []byte
}

func NewTokenValidator(secret []byte) *TokenValidator {
	return &TokenValidator{secret: secret}
}

func (tv *TokenValidator) ParseAndValidate(tokenStr string) (*DeviceClaims, error) {
	tokenStr = strings.TrimSpace(tokenStr)
	tokenStr = strings.TrimPrefix(tokenStr, "Bearer ")
	if tokenStr == "" {
		return nil, jwt.ErrTokenMalformed
	}

	claims := &DeviceClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return tv.secret, nil
	})

	if err != nil || !token.Valid {
		return nil, errors.New("Invalid or expired token")
	}

	if claims.ExpiresAt != nil && claims.ExpiresAt.Before(time.Now()) {
		return nil, errors.New("Token expired")
	}

	return claims, nil
}
