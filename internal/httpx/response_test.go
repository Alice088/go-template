package httpx

import (
	"encoding/json"
	"errors"
	"go-template/internal/apierr"
	"go-template/internal/dto"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	JSON(rec, http.StatusCreated, map[string]string{"status": "ok"})

	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want 201", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != contentTypeJSON {
		t.Errorf("content type = %q, want %q", got, contentTypeJSON)
	}
}

func TestJSONNilPayload(t *testing.T) {
	rec := httptest.NewRecorder()
	JSON(rec, http.StatusNoContent, nil)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty", rec.Body.String())
	}
}

func TestErrorRendersPublicError(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/things", nil)

	Error(rec, req, apierr.NotFound("thing 42 does not exist"))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}

	var body dto.ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Error.Code != "not_found" {
		t.Errorf("code = %q, want not_found", body.Error.Code)
	}
	if body.Error.Message != "not found" {
		t.Errorf("message = %q, want not found", body.Error.Message)
	}
}

func TestErrorHidesUnknownError(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/things", nil)

	Error(rec, req, errors.New("pq: relation \"users\" does not exist"))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}

	var body dto.ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Error.Code != "internal_error" {
		t.Errorf("code = %q, want internal_error", body.Error.Code)
	}
	if body.Error.Message != "internal server error" {
		t.Errorf("message = %q, must not leak internals", body.Error.Message)
	}
	if body.Error.Message == "" || strings.Contains(body.Error.Message, "relation") {
		t.Errorf("message %q leaks the database error", body.Error.Message)
	}
}

func TestErrorIncludesValidationDetails(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/users", nil)

	Error(rec, req, apierr.Validation("invalid payload", map[string]string{"email": "is required"}))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}

	var body dto.ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Error.Code != "validation_failed" {
		t.Errorf("code = %q, want validation_failed", body.Error.Code)
	}
	if body.Error.Details["email"] != "is required" {
		t.Errorf("details = %v, want email problem", body.Error.Details)
	}
}

func TestErrorIgnoresNil(t *testing.T) {
	rec := httptest.NewRecorder()
	Error(rec, httptest.NewRequest(http.MethodGet, "/", nil), nil)

	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty", rec.Body.String())
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want the untouched 200", rec.Code)
	}
}

func TestNoContent(t *testing.T) {
	rec := httptest.NewRecorder()
	NoContent(rec)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", rec.Code)
	}
}
