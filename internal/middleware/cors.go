package middleware

import (
	"net/http"
	"slices"
)

// CORS returns a middleware that answers preflight requests and adds CORS
// headers for the allowed origins. A single "*" allows any origin.
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	allowAll := slices.Contains(allowedOrigins, "*")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			allowed := origin != "" && (allowAll || slices.Contains(allowedOrigins, origin))

			if allowed {
				if allowAll {
					w.Header().Set("Access-Control-Allow-Origin", "*")
				} else {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Add("Vary", "Origin")
				}
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-Request-Id")
				w.Header().Set("Access-Control-Max-Age", "600")
			}

			// Only a real preflight request is terminated here; a plain OPTIONS
			// request is handed over to the router.
			if allowed && r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
