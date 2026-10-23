package main

import (
	"CentralServer/internal/auth"
	"CentralServer/internal/handler"
	"CentralServer/internal/telemetry"
	"CentralServer/internal/upstream"
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"CentralServer/internal/config"
	"CentralServer/internal/store"
	"CentralServer/utils"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Storage & Outbound Clients
	mongoDB, err := store.NewMongoDB(ctx, cfg.MongoURI, cfg.MongoDBName)
	if err != nil {
		utils.Logger{}.Error("MongoDB connection failed: " + err.Error())
		os.Exit(1)
	}
	defer func() {
		shutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = mongoDB.Close(shutCtx)
	}()

	telemetryRepo, err := store.NewTelemetryRepo(ctx, mongoDB)
	if err != nil {
		utils.Logger{}.Error("Telemetry repository initialization failed: " + err.Error())
		os.Exit(1)
	}

	aiClient := upstream.NewAIClient(cfg.AIServerURL)
	validator := auth.NewTokenValidator(cfg.JWTSecret)

	// Hub Worker
	hub := telemetry.NewHub(telemetryRepo, aiClient, 10000)
	hub.Start(4)
	defer hub.Stop()

	// Routing
	mux := http.NewServeMux()
	mux.HandleFunc("GET /ws/sensors", handler.HandleSensorWS(hub, validator))
	mux.Handle("POST /api/v1/devices/image", handler.NewImageHandler(aiClient, validator))

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go func() {
		utils.Logger{}.Info("Listening on port: " + cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			utils.Logger{}.Error("HTTP listen failed: " + err.Error())
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	utils.Logger{}.Info("Shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}
