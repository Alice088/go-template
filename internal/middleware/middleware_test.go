package middleware

import (
	"context"
	"encoding/json"
	"go-template/internal/auth"
	"go-template/internal/dto"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

func TestLoggingSetsRequestIDHeader(t *testing.T) {
	router := chi.NewRouter()
	router.Use(chimw.RequestID)
	router.Use(Logging)
	router.Get("/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusTeapot {
		t.Errorf("status = %d, want 418", rec.Code)
	}
	if got := rec.Header().Get(chimw.RequestIDHeader); got == "" {
		t.Error("request ID header must be echoed back")
	}
}

func TestRecoveryRendersJSON(t *testing.T) {
	handler := Recovery(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

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
	if strings.Contains(body.Error.Message, "boom") {
		t.Errorf("message %q leaks the panic value", body.Error.Message)
	}
}

func TestRecoveryPassesThroughWithoutPanic(t *testing.T) {
	handler := Recovery(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusAccepted {
		t.Errorf("status = %d, want 202", rec.Code)
	}
}

func TestCORSSetsHeadersForAllowedOrigin(t *testing.T) {
	handler := CORS([]string{"https://app.example"})(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://app.example")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example" {
		t.Errorf("allow origin = %q, want https://app.example", got)
	}
	if got := rec.Header().Get("Vary"); got != "Origin" {
		t.Errorf("vary = %q, want Origin", got)
	}
}

func TestCORSRejectsForeignOrigin(t *testing.T) {
	handler := CORS([]string{"https://app.example"})(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://evil.example")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("allow origin = %q, want empty", got)
	}
}

func TestJWTAuthRejectsMissingHeader(t *testing.T) {
	manager := auth.NewJWTManager("test-secret-key-that-is-long-enough", time.Minute, time.Hour)
	handler := JWTAuth(manager)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Errorf("content type = %q, want json", got)
	}
}

func TestJWTAuthRejectsMalformedHeader(t *testing.T) {
	manager := auth.NewJWTManager("test-secret-key-that-is-long-enough", time.Minute, time.Hour)
	handler := JWTAuth(manager)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Token abc")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestJWTAuthPassesClaimsToContext(t *testing.T) {
	manager := auth.NewJWTManager("test-secret-key-that-is-long-enough", time.Minute, time.Hour)
	token, err := manager.GenerateAccessToken(99, "admin")
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	var gotUserID int64
	var gotRole string

	handler := JWTAuth(manager)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		gotUserID = GetUserID(r.Context())
		gotRole = GetRole(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	handler.ServeHTTP(httptest.NewRecorder(), req)

	if gotUserID != 99 {
		t.Errorf("user id = %d, want 99", gotUserID)
	}
	if gotRole != "admin" {
		t.Errorf("role = %q, want admin", gotRole)
	}
}

func TestRequireRole(t *testing.T) {
	allowed := RequireRole("admin")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	adminCtx := context.WithValue(context.Background(), RoleKey, "admin")
	adminReq := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(adminCtx)
	adminRec := httptest.NewRecorder()
	allowed.ServeHTTP(adminRec, adminReq)
	if adminRec.Code != http.StatusOK {
		t.Errorf("admin status = %d, want 200", adminRec.Code)
	}

	userCtx := context.WithValue(context.Background(), RoleKey, "user")
	userReq := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(userCtx)
	userRec := httptest.NewRecorder()
	allowed.ServeHTTP(userRec, userReq)
	if userRec.Code != http.StatusForbidden {
		t.Errorf("user status = %d, want 403", userRec.Code)
	}
}

func TestGetUserIDWithoutValue(t *testing.T) {
	if got := GetUserID(context.Background()); got != 0 {
		t.Errorf("user id = %d, want 0", got)
	}
	if got := GetRole(context.Background()); got != "" {
		t.Errorf("role = %q, want empty", got)
	}
}

func TestTimeoutSetsDeadline(t *testing.T) {
	var hasDeadline bool
	var remaining time.Duration

	handler := Timeout(2 * time.Second)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		deadline, ok := r.Context().Deadline()
		hasDeadline = ok
		if ok {
			remaining = time.Until(deadline)
		}
	}))

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	if !hasDeadline {
		t.Fatal("the request context must carry a deadline")
	}
	if remaining <= 0 || remaining > 2*time.Second {
		t.Errorf("remaining = %s, want (0, 2s]", remaining)
	}
}
