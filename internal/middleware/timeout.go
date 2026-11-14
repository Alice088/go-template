package middleware

import (
	"context"
	"net/http"
	"time"
)

// Timeout bounds the lifetime of the request context. It does not write a
// response itself: handlers are expected to respect r.Context() and return a
// JSON error (usually apierr.Unavailable) when the deadline is exceeded. That
// keeps every response, including timeouts, in the same JSON format.
func Timeout(d time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), d)
			defer cancel()

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
