// Package v1 provides the API v1 router with middleware.
package v1

import (
	"go-template/internal/handler"
	"go-template/internal/middleware"
	"net/http"

	httpSwagger "github.com/swaggo/http-swagger/v2"
)

// NewRouter builds the v1 HTTP handler with all routes and middleware.
func NewRouter(healthHandler *handler.HealthHandler) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("GET /swagger/", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	mux.HandleFunc("GET /health", healthHandler.Health)

	var h http.Handler = mux
	h = middleware.Logging(h)
	h = middleware.Recovery(h)

	return h
}
