// Package web provides Postal's standard HTTP response envelope and the central
// error-rendering logic shared by every handler. It is the one place that knows
// how to turn an apperr.Error into an HTTP status and a safe client payload, so
// every endpoint returns a consistent shape.
package web

import (
	"encoding/json"
	"net/http"
)

// Envelope is the standard success response shape: {"data": ...}. Meta is
// omitted unless set (pagination, etc.).
type Envelope struct {
	Data any            `json:"data"`
	Meta map[string]any `json:"meta,omitempty"`
}

// ErrorEnvelope is the standard error response shape: {"error": {...}}.
type ErrorEnvelope struct {
	Error ErrorBody `json:"error"`
}

// ErrorBody is the client-facing error detail. It never contains internal
// causes — only a stable code, a safe message, optional field errors, and the
// request ID for support correlation.
type ErrorBody struct {
	Code      string      `json:"code"`
	Message   string      `json:"message"`
	Fields    []FieldItem `json:"fields,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
}

// FieldItem is a field-level validation problem in the error envelope.
type FieldItem struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Respond writes data as a standard success envelope with the given status.
func Respond(w http.ResponseWriter, status int, data any) {
	writeJSON(w, status, Envelope{Data: data})
}

// RespondMeta writes a success envelope including a meta object.
func RespondMeta(w http.ResponseWriter, status int, data any, meta map[string]any) {
	writeJSON(w, status, Envelope{Data: data, Meta: meta})
}

// writeJSON encodes v as JSON with the given status code and content type.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
