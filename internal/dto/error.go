package dto

// ErrorResponse is the JSON envelope for every API error.
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

// ErrorBody describes a single API error.
type ErrorBody struct {
	// Code is a stable machine-readable code, e.g. "validation_failed".
	Code string `json:"code"`
	// Message is a human-readable, client-safe message.
	Message string `json:"message"`
	// Details carries field-level problems, e.g. validation failures.
	Details map[string]string `json:"details,omitempty"`
}
