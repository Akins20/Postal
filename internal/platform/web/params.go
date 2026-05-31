package web

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Akins20/postal/internal/platform/apperr"
)

// PathUUID parses a UUID-valued chi route parameter, returning a validation
// error (code "invalid_<name>") when it is missing or malformed. Shared so
// every handler parses path IDs consistently.
func PathUUID(r *http.Request, name string) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, name))
	if err != nil {
		return uuid.Nil, apperr.Validation("invalid_"+name, "invalid "+name)
	}
	return id, nil
}

// TimeRange parses the ?from= and ?to= RFC3339 query parameters, falling back to
// defFrom/defTo when a parameter is absent. Shared so every range endpoint
// validates time windows consistently.
func TimeRange(r *http.Request, defFrom, defTo time.Time) (from, to time.Time, err error) {
	from, to = defFrom, defTo
	if v := r.URL.Query().Get("from"); v != "" {
		from, err = time.Parse(time.RFC3339, v)
		if err != nil {
			return time.Time{}, time.Time{}, apperr.Validation("invalid_from", "from must be RFC3339")
		}
	}
	if v := r.URL.Query().Get("to"); v != "" {
		to, err = time.Parse(time.RFC3339, v)
		if err != nil {
			return time.Time{}, time.Time{}, apperr.Validation("invalid_to", "to must be RFC3339")
		}
	}
	return from, to, nil
}
