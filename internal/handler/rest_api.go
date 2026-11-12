//
// Created by HuyN on 2026/11/12.
// Description: Cung cấp các endpoint HTTP REST phục vụ truy vấn dữ liệu lịch sử cảm biến cho biểu đồ
// và đăng ký mới thiết bị kèm cấp phát JWT token.
//

package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"CentralServer/internal/store"
)

type RestHandler struct {
	telemetryRepo *store.TelemetryRepo
	deviceRepo    *store.DeviceRepo
}

func NewRestHandler(tRepo *store.TelemetryRepo, dRepo *store.DeviceRepo) *RestHandler {
	return &RestHandler{
		telemetryRepo: tRepo,
		deviceRepo:    dRepo,
	}
}

// GetRecentTelemetry trả về dữ liệu lịch sử cho biểu đồ Next.js
func (h *RestHandler) GetRecentTelemetry(w http.ResponseWriter, r *http.Request) {
	deviceID := r.URL.Query().Get("device_id")
	if deviceID == "" {
		deviceID = "plant_node_01" // Mặc định nếu frontend chưa truyền
	}

	limit := int64(50)
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if parsed, err := strconv.ParseInt(lStr, 10, 64); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	data, err := h.telemetryRepo.GetRecentByDevice(r.Context(), deviceID, limit)
	if err != nil {
		http.Error(w, "Lỗi truy vấn dữ liệu", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(data)
}

// RegisterDevice đăng ký thiết bị mới và trả về JWT token
func (h *RestHandler) RegisterDevice(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DeviceID string `json:"device_id"`
		Name     string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Dữ liệu JSON không hợp lệ", http.StatusBadRequest)
		return
	}

	token, err := h.deviceRepo.RegisterDevice(r.Context(), req.DeviceID, req.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"device_id": req.DeviceID,
		"token":     token,
	})
}
