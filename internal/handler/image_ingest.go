package handler

import (
	"io"
	"net/http"

	"CentralServer/internal/auth"
	"CentralServer/internal/upstream"
)

type ImageHandler struct {
	ai        *upstream.AIClient
	validator *auth.TokenValidator
}

func NewImageHandler(ai *upstream.AIClient, validator *auth.TokenValidator) *ImageHandler {
	return &ImageHandler{
		ai:        ai,
		validator: validator,
	}
}

func (h *ImageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	claims, err := h.validator.ParseAndValidate(r.Header.Get("Authorization"))
	if err != nil {
		http.Error(w, "Unauthorized "+err.Error(), http.StatusUnauthorized)
		return
	}

	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	r.Body = http.MaxBytesReader(w, r.Body, 15<<20) // Limit to 15MB on request body

	aiBody, statusCode, err := h.ai.ForwardImageStream(r.Context(), claims.DeviceID, contentType, r.Body)
	if err != nil {
		http.Error(w, "AI server unreachable: "+err.Error(), statusCode)
		return
	}
	defer aiBody.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_, _ = io.Copy(w, aiBody)
}
