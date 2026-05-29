package web

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/Akins20/postal/internal/platform/apperr"
)

// DefaultMaxBodyBytes bounds request bodies by default (1 MiB). Oversized
// bodies are an abuse vector, so decoding is always capped.
const DefaultMaxBodyBytes int64 = 1 << 20

// DecodeJSON reads and strictly decodes the request body into dst, enforcing a
// size limit and rejecting unknown fields and trailing data. It returns a
// validation apperr on malformed input so handlers can return it directly.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	return DecodeJSONLimit(w, r, dst, DefaultMaxBodyBytes)
}

// DecodeJSONLimit is DecodeJSON with an explicit maximum body size in bytes.
func DecodeJSONLimit(w http.ResponseWriter, r *http.Request, dst any, maxBytes int64) error {
	if ct := r.Header.Get("Content-Type"); ct != "" && !strings.HasPrefix(ct, "application/json") {
		return apperr.Validation("unsupported_media_type", "request body must be application/json")
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		return decodeError(err)
	}
	// Reject trailing data after the first JSON value.
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return apperr.Validation("malformed_json", "request body must contain a single JSON object")
	}
	return nil
}

// decodeError maps json decoding failures to safe, specific validation errors.
func decodeError(err error) error {
	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		return apperr.Validation("body_too_large", "request body is too large")
	}

	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		return apperr.Validation("malformed_json", "request body contains malformed JSON")
	}

	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		return apperr.Validation("invalid_field_type", "a field has an invalid type").
			WithField(typeErr.Field, "invalid type")
	}

	if errors.Is(err, io.EOF) {
		return apperr.Validation("empty_body", "request body must not be empty")
	}

	if strings.HasPrefix(err.Error(), "json: unknown field ") {
		field := strings.TrimSuffix(strings.TrimPrefix(err.Error(), "json: unknown field \""), "\"")
		return apperr.Validation("unknown_field", "request contains an unknown field").
			WithField(field, "unknown field")
	}

	return apperr.Validation("malformed_json", "request body could not be parsed")
}
