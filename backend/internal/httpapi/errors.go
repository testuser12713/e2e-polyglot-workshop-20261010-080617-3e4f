package httpapi

import (
	"encoding/json"
	"net/http"
)

// Machine-readable error codes shared by every failure response.
const (
	CodeValidationError   = "validation_error"
	CodeNotFound          = "not_found"
	CodeConflict          = "conflict"
	CodeInvalidTransition = "invalid_transition"
	CodeUnauthorized      = "unauthorized"
	CodeRateLimited       = "rate_limited"
	CodeInternalError     = "internal_error"
	// CodeNotImplemented marks a route that is registered but whose feature
	// ticket has not landed yet. It answers 501, never 500.
	CodeNotImplemented = "not_implemented"
)

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type errorEnvelope struct {
	Error apiError `json:"error"`
}

// WriteJSON writes v as a JSON response with the given status code.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(v)
}

// WriteError writes the shared error envelope: {"error":{"code","message"}}.
func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, errorEnvelope{Error: apiError{Code: code, Message: message}})
}

// NotYetImplemented answers 501 for a route whose feature ticket has not landed
// yet. It is a declared placeholder, never a panic and never a 500.
func NotYetImplemented(w http.ResponseWriter, _ *http.Request) {
	WriteError(w, http.StatusNotImplemented, CodeNotImplemented, "not implemented yet")
}
