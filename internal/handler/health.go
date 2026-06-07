package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"go-template/internal/dto"
)

type HealthHandler struct {
	version string
}

func NewHealthHandler(version string) *HealthHandler {
	return &HealthHandler{version: version}
}

func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	resp := dto.HealthResponse{
		Status:    "ok",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Version:   h.version,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
