package web

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/Akins20/postal/internal/platform/apperr"
)

// statusByKind maps each apperr.Kind to its HTTP status. Centralizing the
// mapping here guarantees every endpoint answers consistently.
var statusByKind = map[apperr.Kind]int{
	apperr.KindValidation:   http.StatusBadRequest,
	apperr.KindUnauthorized: http.StatusUnauthorized,
	apperr.KindForbidden:    http.StatusForbidden,
	apperr.KindNotFound:     http.StatusNotFound,
	apperr.KindConflict:     http.StatusConflict,
	apperr.KindRateLimited:  http.StatusTooManyRequests,
	apperr.KindInternal:     http.StatusInternalServerError,
}

// HandlerFunc is a handler that may return an error. Returning an error lets
// handlers stay terse (return apperr.Validation(...)) and routes all error
// rendering through one place.
type HandlerFunc func(http.ResponseWriter, *http.Request) error

// Handler adapts a fallible HandlerFunc into an http.HandlerFunc, rendering any
// returned error via Fail. log is used to record internal (5xx) errors at the
// boundary — the single place errors are logged, per the coding standards.
func Handler(log *slog.Logger, h HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			Fail(w, r, log, err)
		}
	}
}

// Fail renders err as the standard error envelope with the mapped HTTP status.
// Client-safe details (code, message, field errors) come from the apperr.Error;
// the underlying cause is logged for 5xx but never sent to the client.
func Fail(w http.ResponseWriter, r *http.Request, log *slog.Logger, err error) {
	reqID := middleware.GetReqID(r.Context())

	ae, ok := apperr.As(err)
	if !ok {
		// Unclassified errors are treated as internal and never leak their text.
		ae = apperr.Internal(err)
	}

	status, found := statusByKind[ae.Kind]
	if !found {
		status = http.StatusInternalServerError
	}

	// Log server-side faults with the cause; client faults are not noise-logged.
	if status >= http.StatusInternalServerError && log != nil {
		log.LogAttrs(r.Context(), slog.LevelError, "request failed",
			slog.String("request_id", reqID),
			slog.String("code", ae.Code),
			slog.String("error", err.Error()),
		)
	}

	body := ErrorEnvelope{Error: ErrorBody{
		Code:      ae.Code,
		Message:   ae.Message,
		Fields:    toFieldItems(ae.Fields),
		RequestID: reqID,
	}}
	writeJSON(w, status, body)
}

// StatusFor returns the HTTP status that err maps to. Useful in tests.
func StatusFor(err error) int {
	if s, ok := statusByKind[apperr.KindOf(err)]; ok {
		return s
	}
	return http.StatusInternalServerError
}

// toFieldItems converts apperr field errors into the envelope representation.
func toFieldItems(in []apperr.FieldError) []FieldItem {
	if len(in) == 0 {
		return nil
	}
	out := make([]FieldItem, len(in))
	for i, f := range in {
		out[i] = FieldItem{Field: f.Field, Message: f.Message}
	}
	return out
}
