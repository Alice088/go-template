// Package httpx provides JSON request/response helpers shared by handlers and
// middleware: one place that knows how a response or an error is written.
package httpx

import (
	"encoding/json"
	"go-template/internal/apierr"
	"go-template/internal/dto"
	"net/http"

	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog/log"
)

const contentTypeJSON = "application/json; charset=utf-8"

// JSON writes payload as JSON with the given status code.
func JSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", contentTypeJSON)
	w.WriteHeader(status)
	if payload == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		// The status code is already sent, so the only thing left is to report.
		log.Error().Err(err).Msg("failed to encode json response")
	}
}

// NoContent writes an empty 204 response.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// Public writes an already resolved public error as JSON, without logging it.
// Use it when the caller has logged the error itself, e.g. in a panic recovery.
func Public(w http.ResponseWriter, pub apierr.PublicError) {
	JSON(w, pub.HTTPStatus(), dto.ErrorResponse{
		Error: dto.ErrorBody{
			Code:    string(pub.Code()),
			Message: pub.PublicMessage(),
			Details: pub.Details(),
		},
	})
}

// Error writes err as a JSON error response.
//
// Errors implementing apierr.PublicError are rendered with their own status,
// code and public message. Any other error is rendered as a generic 500, and
// the original text is only logged.
func Error(w http.ResponseWriter, r *http.Request, err error) {
	if err == nil {
		return
	}

	pub, known := apierr.As(err)
	if !known {
		pub = apierr.Internal("unhandled error").WithCause(err)
	}

	event := log.Warn
	if pub.HTTPStatus() >= http.StatusInternalServerError {
		event = log.Error
	}

	event().
		Err(err).
		Str("request_id", chimw.GetReqID(r.Context())).
		Str("method", r.Method).
		Str("path", r.URL.Path).
		Str("code", string(pub.Code())).
		Int("status", pub.HTTPStatus()).
		Msg("request failed")

	Public(w, pub)
}
