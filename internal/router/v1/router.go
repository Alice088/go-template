package v1

import (
	"net/http"

	"go-template/internal/handler"
	"go-template/internal/middleware"
)

func NewRouter(healthHandler *handler.HealthHandler) http.Handler {
	mux := http.NewServeMux()

	// Health check (public)
	mux.HandleFunc("GET /health", healthHandler.Health)

	// Apply global middleware
	var h http.Handler = mux
	h = middleware.Logging(h)
	h = middleware.Recovery(h)

	return h
}
