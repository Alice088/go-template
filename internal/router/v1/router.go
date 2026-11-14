// Package v1 provides the API v1 router with middleware.
package v1

import (
	"go-template/internal/config"
	"go-template/internal/handler"
	"go-template/internal/middleware"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	// Blank import registers the generated OpenAPI document. Without it
	// /swagger/doc.json answers 500, because swaggo cannot find the spec.
	_ "go-template/docs"
)

// NewRouter builds the v1 chi router with all routes and middleware.
func NewRouter(healthHandler *handler.HealthHandler, cfg *config.Config) http.Handler {
	r := chi.NewRouter()

	// Order matters: the request ID must exist before logging, and panics must
	// be caught before they reach the outer net/http server.
	r.Use(chimw.RequestID)
	r.Use(middleware.Logging)
	r.Use(middleware.Recovery)
	r.Use(middleware.CORS(cfg.Server.AllowedOrigins))
	r.Use(chimw.RequestSize(cfg.Server.MaxBodyBytes))
	r.Use(middleware.Timeout(cfg.Server.RequestTimeout))

	// Default 404/405 answers are JSON, like every other error.
	r.NotFound(notFound)
	r.MethodNotAllowed(methodNotAllowed)

	r.Handle("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	r.Route("/v1", func(r chi.Router) {
		r.Get("/health", healthHandler.Health)
		r.Get("/ready", healthHandler.Ready)

		// Register authenticated routes here, for example:
		//
		//	r.Group(func(r chi.Router) {
		//		r.Use(middleware.JWTAuth(jwtManager))
		//		r.Get("/me", userHandler.Me)
		//	})
	})

	return r
}
