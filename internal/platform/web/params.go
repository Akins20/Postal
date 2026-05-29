package web

import (
	"net/http"

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
