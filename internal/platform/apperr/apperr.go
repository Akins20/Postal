// Package apperr defines Postal's domain-neutral error taxonomy. Services and
// stores return *apperr.Error values describing what kind of failure occurred;
// the web layer (internal/platform/web) maps each Kind to an HTTP status and a
// safe client message. Keeping these errors free of net/http preserves the
// handler -> service -> store layering: lower layers never import HTTP.
package apperr

import (
	"errors"
	"fmt"
)

// Kind classifies an error so the HTTP layer and retry logic can treat it
// consistently. The zero value is KindInternal so an uninitialized Error never
// leaks as a success.
type Kind int

// Error kinds. Each maps to one HTTP status in the web layer.
const (
	KindInternal Kind = iota
	KindValidation
	KindUnauthorized
	KindForbidden
	KindNotFound
	KindConflict
	KindRateLimited
)

// String returns a stable, lowercase name for the kind (used in logs).
func (k Kind) String() string {
	switch k {
	case KindValidation:
		return "validation"
	case KindUnauthorized:
		return "unauthorized"
	case KindForbidden:
		return "forbidden"
	case KindNotFound:
		return "not_found"
	case KindConflict:
		return "conflict"
	case KindRateLimited:
		return "rate_limited"
	default:
		return "internal"
	}
}

// FieldError is a single field-level validation problem, surfaced to clients so
// they can highlight the offending input.
type FieldError struct {
	// Field is the offending input field (e.g. "email").
	Field string `json:"field"`
	// Message is a safe, human-readable description of the problem.
	Message string `json:"message"`
}

// Error is Postal's standard application error. Code is a stable machine-
// readable string (e.g. "invalid_email"); Message is safe to show clients;
// wrapped holds the underlying cause for logs and errors.Is/As inspection.
type Error struct {
	Kind    Kind
	Code    string
	Message string
	Fields  []FieldError
	wrapped error
}

// Error implements the error interface, including the wrapped cause when present.
func (e *Error) Error() string {
	if e.wrapped != nil {
		return fmt.Sprintf("%s: %s: %v", e.Kind, e.Code, e.wrapped)
	}
	return fmt.Sprintf("%s: %s: %s", e.Kind, e.Code, e.Message)
}

// Unwrap exposes the wrapped cause for errors.Is / errors.As.
func (e *Error) Unwrap() error { return e.wrapped }

// WithField appends a field-level validation detail and returns the error so
// constructors can be chained.
func (e *Error) WithField(field, message string) *Error {
	e.Fields = append(e.Fields, FieldError{Field: field, Message: message})
	return e
}

// New builds an Error of the given kind. Prefer the kind-specific constructors
// below for readability.
func New(kind Kind, code, message string) *Error {
	return &Error{Kind: kind, Code: code, Message: message}
}

// Wrap annotates an existing error with a kind, code, and safe message. The
// original error is preserved for logging and errors.Is/As.
func Wrap(err error, kind Kind, code, message string) *Error {
	return &Error{Kind: kind, Code: code, Message: message, wrapped: err}
}

// Validation builds a 400-class error.
func Validation(code, message string) *Error { return New(KindValidation, code, message) }

// Unauthorized builds a 401-class error (missing/invalid credentials).
func Unauthorized(code, message string) *Error { return New(KindUnauthorized, code, message) }

// Forbidden builds a 403-class error (authenticated but not permitted).
func Forbidden(code, message string) *Error { return New(KindForbidden, code, message) }

// NotFound builds a 404-class error.
func NotFound(code, message string) *Error { return New(KindNotFound, code, message) }

// Conflict builds a 409-class error (e.g. duplicate resource).
func Conflict(code, message string) *Error { return New(KindConflict, code, message) }

// RateLimited builds a 429-class error.
func RateLimited(code, message string) *Error { return New(KindRateLimited, code, message) }

// Internal builds a 500-class error wrapping the underlying cause. The message
// is intentionally generic; the cause is logged, never shown to clients.
func Internal(err error) *Error {
	return &Error{Kind: KindInternal, Code: "internal_error", Message: "an internal error occurred", wrapped: err}
}

// KindOf reports the Kind of err if it is (or wraps) an *Error, else KindInternal.
func KindOf(err error) Kind {
	var e *Error
	if errors.As(err, &e) {
		return e.Kind
	}
	return KindInternal
}

// As extracts the first *Error in err's chain, reporting whether one was found.
func As(err error) (*Error, bool) {
	var e *Error
	ok := errors.As(err, &e)
	return e, ok
}
