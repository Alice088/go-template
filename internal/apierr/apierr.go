// Package apierr defines application errors.
//
// Every error carries two messages:
//
//   - Message is the internal text meant for logs and traces. It may contain
//     details that must never reach a client (SQL, paths, identifiers).
//   - PublicMessage is the text that is safe to render into an HTTP response.
//
// Errors that implement PublicError can be serialized to the client as JSON.
// Any other error is treated as an internal failure and rendered as a generic
// 500 response, while the real cause is only logged.
package apierr

import (
	"errors"
	"fmt"
	"net/http"
)

// Code is a stable machine-readable error code returned to API clients.
// Clients may branch on it; it is part of the API contract.
type Code string

// Error codes returned to API clients.
const (
	CodeInternal         Code = "internal_error"
	CodeBadRequest       Code = "bad_request"
	CodeValidation       Code = "validation_failed"
	CodeUnauthorized     Code = "unauthorized"
	CodeForbidden        Code = "forbidden"
	CodeNotFound         Code = "not_found"
	CodeMethodNotAllowed Code = "method_not_allowed"
	CodeConflict         Code = "conflict"
	CodeUnavailable      Code = "unavailable"
)

// Default public messages per code. They are intentionally generic: they tell
// the client what happened without leaking internals.
var defaultPublicMessages = map[Code]string{
	CodeInternal:         "internal server error",
	CodeBadRequest:       "bad request",
	CodeValidation:       "validation failed",
	CodeUnauthorized:     "unauthorized",
	CodeForbidden:        "forbidden",
	CodeNotFound:         "not found",
	CodeMethodNotAllowed: "method not allowed",
	CodeConflict:         "conflict",
	CodeUnavailable:      "service unavailable",
}

// PublicError is an error that is safe to render to API clients.
//
// Use errors.As with a variable of this type to detect such errors:
//
//	var pub apierr.PublicError
//	if errors.As(err, &pub) { ... }
type PublicError interface {
	error
	// HTTPStatus is the status code to send to the client.
	HTTPStatus() int
	// Code is the stable machine-readable error code.
	Code() Code
	// PublicMessage is the client-facing message, safe to expose.
	PublicMessage() string
	// Details carries optional field-level information, e.g. validation errors.
	Details() map[string]string
}

// Error is the base application error. It implements PublicError, so every
// application error is safe to expose — because its public message is chosen
// explicitly at construction time.
type Error struct {
	code    Code
	status  int
	message string
	public  string
	details map[string]string
	cause   error
}

// Compile-time assertion that *Error satisfies PublicError.
var _ PublicError = (*Error)(nil)

// New creates an application error with an explicit status and public message.
// Prefer the named constructors below; New is for unusual cases.
func New(code Code, status int, message, public string) *Error {
	if public == "" {
		public = defaultPublicMessages[code]
	}
	if public == "" {
		public = http.StatusText(status)
	}
	return &Error{
		code:    code,
		status:  status,
		message: message,
		public:  public,
	}
}

// Internal reports an unexpected failure. The public message stays generic.
func Internal(message string) *Error {
	return New(CodeInternal, http.StatusInternalServerError, message, "")
}

// BadRequest reports a malformed or unprocessable request.
func BadRequest(message string) *Error {
	return New(CodeBadRequest, http.StatusBadRequest, message, "")
}

// Unauthorized reports missing or invalid credentials.
func Unauthorized(message string) *Error {
	return New(CodeUnauthorized, http.StatusUnauthorized, message, "")
}

// Forbidden reports insufficient permissions.
func Forbidden(message string) *Error {
	return New(CodeForbidden, http.StatusForbidden, message, "")
}

// NotFound reports a missing resource.
func NotFound(message string) *Error {
	return New(CodeNotFound, http.StatusNotFound, message, "")
}

// MethodNotAllowed reports a route that exists but does not support the method.
func MethodNotAllowed(message string) *Error {
	return New(CodeMethodNotAllowed, http.StatusMethodNotAllowed, message, "")
}

// Conflict reports a state conflict, e.g. a duplicate unique value.
func Conflict(message string) *Error {
	return New(CodeConflict, http.StatusConflict, message, "")
}

// Unavailable reports an unavailable dependency, e.g. a database outage.
func Unavailable(message string) *Error {
	return New(CodeUnavailable, http.StatusServiceUnavailable, message, "")
}

// WithCause returns a copy of the error that wraps cause.
// The cause is logged and never exposed to clients.
func (e *Error) WithCause(cause error) *Error {
	clone := *e
	clone.cause = cause
	return &clone
}

// WithDetails returns a copy of the error with field-level details attached.
func (e *Error) WithDetails(details map[string]string) *Error {
	clone := *e
	clone.details = details
	return &clone
}

// WithPublic returns a copy of the error with a custom public message.
// Use it only for messages that are safe to expose.
func (e *Error) WithPublic(public string) *Error {
	clone := *e
	clone.public = public
	return &clone
}

// Error implements the error interface. The text contains the internal
// message and is intended for logs only.
func (e *Error) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.code, e.message, e.cause)
	}
	return fmt.Sprintf("%s: %s", e.code, e.message)
}

// Unwrap exposes the wrapped cause to errors.Is and errors.As.
func (e *Error) Unwrap() error { return e.cause }

// HTTPStatus implements PublicError.
func (e *Error) HTTPStatus() int { return e.status }

// Code implements PublicError.
func (e *Error) Code() Code { return e.code }

// PublicMessage implements PublicError.
func (e *Error) PublicMessage() string { return e.public }

// Details implements PublicError.
func (e *Error) Details() map[string]string { return e.details }

// ValidationError is a BadRequest error that carries per-field problems.
//
//	var err error = apierr.Validation("invalid signup payload", map[string]string{
//		"email":    "is required",
//		"password": "is too short",
//	})
//
// It is not built by embedding Error, because the promoted Error() method
// would collide with the embedded field name.
type ValidationError struct {
	base   *Error
	Fields map[string]string
}

// Compile-time assertion that *ValidationError satisfies PublicError.
var _ PublicError = (*ValidationError)(nil)

// Validation creates a validation error with per-field details.
func Validation(message string, fields map[string]string) *ValidationError {
	return &ValidationError{
		base:   New(CodeValidation, http.StatusBadRequest, message, ""),
		Fields: fields,
	}
}

// Error implements the error interface with the internal message.
func (v *ValidationError) Error() string { return v.base.Error() }

// Unwrap exposes the wrapped cause.
func (v *ValidationError) Unwrap() error { return v.base.Unwrap() }

// HTTPStatus implements PublicError.
func (v *ValidationError) HTTPStatus() int { return v.base.HTTPStatus() }

// Code implements PublicError.
func (v *ValidationError) Code() Code { return v.base.Code() }

// PublicMessage implements PublicError.
func (v *ValidationError) PublicMessage() string { return v.base.PublicMessage() }

// Details implements PublicError and returns the per-field problems.
func (v *ValidationError) Details() map[string]string { return v.Fields }

// WithCause returns a copy of the error that wraps cause.
func (v *ValidationError) WithCause(cause error) *ValidationError {
	return &ValidationError{base: v.base.WithCause(cause), Fields: v.Fields}
}

// WithPublic returns a copy of the error with a custom public message.
func (v *ValidationError) WithPublic(public string) *ValidationError {
	return &ValidationError{base: v.base.WithPublic(public), Fields: v.Fields}
}

// As reports whether err (or anything it wraps) implements PublicError.
func As(err error) (PublicError, bool) {
	if pub, ok := errors.AsType[PublicError](err); ok {
		return pub, true
	}
	return nil, false
}

// ToPublic converts any error into a PublicError. Known application errors are
// returned unchanged; anything else becomes a generic internal error that keeps
// the original cause for logging.
func ToPublic(err error) PublicError {
	if err == nil {
		return nil
	}
	if pub, ok := As(err); ok {
		return pub
	}
	return Internal("unhandled error").WithCause(err)
}
