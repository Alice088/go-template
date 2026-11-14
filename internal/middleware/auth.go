package middleware

import (
	"context"
	"go-template/internal/apierr"
	"go-template/internal/auth"
	"go-template/internal/httpx"
	"net/http"
	"strings"
)

type contextKey string

const (
	// UserIDKey is the context key for the authenticated user's ID.
	UserIDKey contextKey = "user_id"
	// RoleKey is the context key for the authenticated user's role.
	RoleKey contextKey = "role"
)

// JWTAuth returns middleware that validates a Bearer JWT token.
func JWTAuth(jwtManager *auth.JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, err := bearerToken(r)
			if err != nil {
				httpx.Error(w, r, err)
				return
			}

			claims, err := jwtManager.ValidateToken(token)
			if err != nil {
				httpx.Error(w, r, apierr.Unauthorized("invalid or expired token").WithCause(err))
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, RoleKey, claims.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// bearerToken extracts the token from the Authorization header.
func bearerToken(r *http.Request) (string, error) {
	header := r.Header.Get("Authorization")
	if header == "" {
		return "", apierr.Unauthorized("missing authorization header")
	}

	scheme, token, found := strings.Cut(header, " ")
	if !found || !strings.EqualFold(scheme, "bearer") || token == "" {
		return "", apierr.Unauthorized("invalid authorization header format")
	}
	return token, nil
}

// RequireRole returns middleware that allows only the given roles.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := GetRole(r.Context())
			if _, ok := allowed[role]; !ok {
				httpx.Error(w, r, apierr.Forbidden("insufficient permissions"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// GetUserID extracts the authenticated user ID from context.
func GetUserID(ctx context.Context) int64 {
	if id, ok := ctx.Value(UserIDKey).(int64); ok {
		return id
	}
	return 0
}

// GetRole extracts the authenticated user role from context.
func GetRole(ctx context.Context) string {
	if role, ok := ctx.Value(RoleKey).(string); ok {
		return role
	}
	return ""
}
