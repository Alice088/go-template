package v1

import (
	"go-template/internal/apierr"
	"go-template/internal/httpx"
	"net/http"
)

// notFound renders an unknown route as a JSON 404.
func notFound(w http.ResponseWriter, r *http.Request) {
	httpx.Error(w, r, apierr.NotFound("route not found"))
}

// methodNotAllowed renders an unsupported method as a JSON 405.
func methodNotAllowed(w http.ResponseWriter, r *http.Request) {
	httpx.Error(w, r, apierr.MethodNotAllowed("method not allowed for route "+r.URL.Path))
}
