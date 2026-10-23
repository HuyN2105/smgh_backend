package config

import (
	"CentralServer/utils"
	"errors"
	"os"
)

type Config struct {
	Port        string
	MongoURI    string
	MongoDBName string
	AIServerURL string
	JWTSecret   []byte
}

func Load() (*Config, error) {
	port := os.Getenv("PORT")
	if port == "" {
		utils.Logger{}.Warn("Failed to parse PORT environment variable, using default port 8080")
		port = "8080"
	}

	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		err := errors.New("MONGO_URI environment variable is not set")
		utils.Logger{}.Error("Failed to parse MONGO_URI environment variable")
		return nil, err
	}

	mongoDBName := os.Getenv("MONGO_DB_NAME")
	if mongoDBName == "" {
		err := errors.New("MONGO_DB_NAME environment variable is not set")
		utils.Logger{}.Error("Failed to parse MONGO_DB_NAME environment variable")
		return nil, err
	}

	AIServerURL := os.Getenv("AI_SERVER_URL")
	if AIServerURL == "" {
		err := errors.New("AI_SERVER_URL environment variable is not set")
		utils.Logger{}.Error("Failed to parse AI_SERVER_URL environment variable")
		return nil, err
	}

	JWTSecret := os.Getenv("JWT_SECRET")
	if JWTSecret == "" {
		err := errors.New("JWT_SECRET environment variable is not set")
		utils.Logger{}.Error("Failed to parse JWT_SECRET environment variable")
		return nil, err
	}

	return &Config{
		Port:        port,
		MongoURI:    mongoURI,
		MongoDBName: mongoDBName,
		AIServerURL: AIServerURL,
		JWTSecret:   []byte(JWTSecret),
	}, nil
}
