//
// Created by HuyN on 2026/10/14.
// Description: Entry point khởi tạo cấu hình, kết nối MongoDB Atlas, khởi chạy Telemetry Hub,
// Dashboard Hub và đăng ký toàn bộ các endpoint HTTP REST & WebSocket cho Central Server.
//

package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"CentralServer/internal/auth"
	"CentralServer/internal/config"
	"CentralServer/internal/handler"
	"CentralServer/internal/store"
	"CentralServer/internal/telemetry"
	"CentralServer/internal/upstream"
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

	// DeviceRepo và RestHandler
	deviceRepo := store.NewDeviceRepo(mongoDB, cfg.JWTSecret)
	restHandler := handler.NewRestHandler(telemetryRepo, deviceRepo)

	// DashboardHub
	dashHub := handler.NewDashboardHub()
	go dashHub.Run()
	dashHandler := handler.NewDashboardHandler(dashHub)

	// Hub Worker
	hub := telemetry.NewHub(telemetryRepo, aiClient, 10000)
	hub.Start(4)
	defer hub.Stop()

	// Routing
	mux := http.NewServeMux()
	mux.HandleFunc("GET /ws/sensors", handler.HandleSensorWS(hub, validator))
	mux.Handle("POST /api/v1/devices/image", handler.NewImageHandler(aiClient, validator))
	mux.HandleFunc("/ws/dashboard", dashHandler.ServeWS)

	// REST API lấy lịch sử & đăng ký thiết bị
	mux.HandleFunc("GET /api/v1/telemetry/recent", restHandler.GetRecentTelemetry)
	mux.HandleFunc("POST /api/v1/devices/register", restHandler.RegisterDevice)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      enableCORS(mux),
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

func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
