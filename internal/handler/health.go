package handler

import (
	"context"
	"go-template/internal/apierr"
	"go-template/internal/dto"
	"go-template/internal/httpx"
	"net/http"
	"time"
)

// readinessTimeout bounds the database check of the readiness endpoint.
const readinessTimeout = 2 * time.Second

// Pinger is the subset of a database connection pool used by the health
// handler. Keeping it an interface makes the handler testable without a real
// database.
type Pinger interface {
	Ping(ctx context.Context) error
}

// HealthHandler serves the liveness and readiness endpoints.
type HealthHandler struct {
	version string
	db      Pinger
}

// NewHealthHandler creates a HealthHandler with the given app version and
// database checker. db may be nil, in which case readiness does not check it.
func NewHealthHandler(version string, db Pinger) *HealthHandler {
	return &HealthHandler{version: version, db: db}
}

// Health godoc
// @Summary      Liveness probe
// @Description  Reports that the process is up. It never touches dependencies.
// @Tags         system
// @Produce      json
// @Success      200  {object}  dto.HealthResponse
// @Router       /health [get]
func (h *HealthHandler) Health(w http.ResponseWriter, _ *http.Request) {
	httpx.JSON(w, http.StatusOK, dto.HealthResponse{
		Status:    "ok",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Version:   h.version,
	})
}

// Ready godoc
// @Summary      Readiness probe
// @Description  Reports whether the service can serve traffic, including its dependencies.
// @Tags         system
// @Produce      json
// @Success      200  {object}  dto.ReadyResponse
// @Failure      503  {object}  dto.ErrorResponse
// @Router       /ready [get]
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	checks := map[string]string{"database": "skipped"}

	if h.db != nil {
		ctx, cancel := context.WithTimeout(r.Context(), readinessTimeout)
		defer cancel()

		if err := h.db.Ping(ctx); err != nil {
			httpx.Error(w, r, apierr.Unavailable("database is not reachable").WithCause(err))
			return
		}
		checks["database"] = "up"
	}

	httpx.JSON(w, http.StatusOK, dto.ReadyResponse{
		Status:    "ok",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Checks:    checks,
	})
}
