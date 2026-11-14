package apierr

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestConstructorsMapToStatuses(t *testing.T) {
	tests := []struct {
		name   string
		err    PublicError
		status int
		code   Code
		public string
	}{
		{"internal", Internal("boom"), http.StatusInternalServerError, CodeInternal, "internal server error"},
		{"bad request", BadRequest("bad"), http.StatusBadRequest, CodeBadRequest, "bad request"},
		{"unauthorized", Unauthorized("nope"), http.StatusUnauthorized, CodeUnauthorized, "unauthorized"},
		{"forbidden", Forbidden("nope"), http.StatusForbidden, CodeForbidden, "forbidden"},
		{"not found", NotFound("gone"), http.StatusNotFound, CodeNotFound, "not found"},
		{"conflict", Conflict("dup"), http.StatusConflict, CodeConflict, "conflict"},
		{"unavailable", Unavailable("down"), http.StatusServiceUnavailable, CodeUnavailable, "service unavailable"},
		{"validation", Validation("bad payload", nil), http.StatusBadRequest, CodeValidation, "validation failed"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.err.HTTPStatus(); got != tc.status {
				t.Errorf("status = %d, want %d", got, tc.status)
			}
			if got := tc.err.Code(); got != tc.code {
				t.Errorf("code = %q, want %q", got, tc.code)
			}
			if got := tc.err.PublicMessage(); got != tc.public {
				t.Errorf("public message = %q, want %q", got, tc.public)
			}
		})
	}
}

func TestInternalMessageIsHidden(t *testing.T) {
	err := Internal("pq: column users.secret does not exist")

	if got := err.Error(); got == "" {
		t.Fatal("internal message must be available for logs")
	}
	if got := err.PublicMessage(); got == err.Error() {
		t.Errorf("public message %q leaks the internal message", got)
	}
}

func TestWithCauseKeepsPublicMessage(t *testing.T) {
	cause := errors.New("connection refused")
	err := Unavailable("database is not reachable").WithCause(cause)

	if !errors.Is(err, cause) {
		t.Error("errors.Is should find the wrapped cause")
	}
	if got := err.PublicMessage(); got != "service unavailable" {
		t.Errorf("public message = %q, want service unavailable", got)
	}
	if got := err.Error(); got == "" || !strings.Contains(got, "connection refused") {
		t.Errorf("internal text %q should include the cause", got)
	}
}

func TestWithCauseDoesNotMutateOriginal(t *testing.T) {
	base := Internal("base")
	_ = base.WithCause(errors.New("cause"))

	if base.Unwrap() != nil {
		t.Error("WithCause must return a copy, leaving the original unchanged")
	}
}

func TestDetails(t *testing.T) {
	err := Validation("invalid payload", map[string]string{"email": "is required"})

	details := err.Details()
	if details["email"] != "is required" {
		t.Errorf("details = %v, want email problem", details)
	}
	if got := err.PublicMessage(); got != "validation failed" {
		t.Errorf("public message = %q, want validation failed", got)
	}

	custom := err.WithPublic("check the highlighted fields")
	if custom.PublicMessage() != "check the highlighted fields" {
		t.Errorf("public message = %q, want the custom text", custom.PublicMessage())
	}
	if err.PublicMessage() != "validation failed" {
		t.Error("WithPublic must not mutate the original error")
	}
}

func TestAs(t *testing.T) {
	inner := NotFound("missing user")

	if _, ok := As(inner); !ok {
		t.Error("As should recognize an application error")
	}

	if _, ok := As(errors.New("plain error")); ok {
		t.Error("As should not recognize a plain error")
	}

	wrapped := fmt.Errorf("handler failed: %w", inner)
	pub, ok := As(wrapped)
	if !ok {
		t.Fatal("As should see through fmt.Errorf wrapping")
	}
	if pub.HTTPStatus() != http.StatusNotFound {
		t.Errorf("status = %d, want 404", pub.HTTPStatus())
	}
}

func TestToPublic(t *testing.T) {
	appErr := Conflict("duplicate email")
	if got := ToPublic(appErr); got != PublicError(appErr) {
		t.Error("ToPublic should return a known application error unchanged")
	}

	plain := errors.New("database exploded")
	pub := ToPublic(plain)
	if pub.HTTPStatus() != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", pub.HTTPStatus())
	}
	if !errors.Is(pub, plain) {
		t.Error("converted error should keep the original cause")
	}
	if pub.PublicMessage() != "internal server error" {
		t.Errorf("public message = %q, want the generic text", pub.PublicMessage())
	}

	if got := ToPublic(nil); got != nil {
		t.Errorf("ToPublic(nil) = %v, want nil", got)
	}
}
