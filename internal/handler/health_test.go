package handler

import (
	"context"
	"encoding/json"
	"errors"
	"go-template/internal/dto"
	"net/http"
	"net/http/httptest"
	"testing"
)

type stubPinger struct{ err error }

func (s stubPinger) Ping(context.Context) error { return s.err }

func TestHealth(t *testing.T) {
	h := NewHealthHandler("1.2.3", nil)

	rec := httptest.NewRecorder()
	h.Health(rec, httptest.NewRequest(http.MethodGet, "/v1/health", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Errorf("content type = %q, want application/json", got)
	}

	var body dto.HealthResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Status != "ok" {
		t.Errorf("status = %q, want ok", body.Status)
	}
	if body.Version != "1.2.3" {
		t.Errorf("version = %q, want 1.2.3", body.Version)
	}
	if body.Timestamp == "" {
		t.Error("timestamp must be set")
	}
}

func TestHealthDoesNotTouchDependencies(t *testing.T) {
	h := NewHealthHandler("1.0.0", stubPinger{err: errors.New("database is down")})

	rec := httptest.NewRecorder()
	h.Health(rec, httptest.NewRequest(http.MethodGet, "/v1/health", nil))

	if rec.Code != http.StatusOK {
		t.Errorf("liveness must stay 200 when a dependency is down, got %d", rec.Code)
	}
}

func TestReadyWithHealthyDatabase(t *testing.T) {
	h := NewHealthHandler("1.0.0", stubPinger{})

	rec := httptest.NewRecorder()
	h.Ready(rec, httptest.NewRequest(http.MethodGet, "/v1/ready", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var body dto.ReadyResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Checks["database"] != "up" {
		t.Errorf("checks = %v, want database up", body.Checks)
	}
}

func TestReadyWithBrokenDatabase(t *testing.T) {
	h := NewHealthHandler("1.0.0", stubPinger{err: errors.New("connection refused")})

	rec := httptest.NewRecorder()
	h.Ready(rec, httptest.NewRequest(http.MethodGet, "/v1/ready", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}

	var body dto.ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Error.Code != "unavailable" {
		t.Errorf("code = %q, want unavailable", body.Error.Code)
	}
	if body.Error.Message == "" || body.Error.Message == "connection refused" {
		t.Errorf("message %q must be public-safe, not the raw cause", body.Error.Message)
	}
}

func TestReadyWithoutDatabase(t *testing.T) {
	h := NewHealthHandler("1.0.0", nil)

	rec := httptest.NewRecorder()
	h.Ready(rec, httptest.NewRequest(http.MethodGet, "/v1/ready", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}
