package middleware

import (
	"fmt"
	"go-template/internal/apierr"
	"go-template/internal/httpx"
	"net/http"
	"runtime/debug"

	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog/log"
)

// Recovery is middleware that catches panics, logs them with a stack trace and
// answers with a generic JSON 500 error.
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}

			log.Error().
				Str("request_id", chimw.GetReqID(r.Context())).
				Interface("panic", rec).
				Bytes("stack", debug.Stack()).
				Msg("panic recovered")

			httpx.Public(w, apierr.Internal("panic recovered").WithCause(fmt.Errorf("%v", rec)))
		}()
		next.ServeHTTP(w, r)
	})
}
