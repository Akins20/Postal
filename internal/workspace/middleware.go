package workspace

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Akins20/postal/internal/platform/apperr"
	"github.com/Akins20/postal/internal/platform/web"
)

// workspaceURLParam is the chi route parameter naming the workspace.
const workspaceURLParam = "workspaceID"

// RequireCapability returns middleware that authorizes the request against a
// workspace capability. It must run after auth's RequireUser (which sets the
// user ID). It resolves the caller's membership for the {workspaceID} route
// param, checks the capability, and stores the membership in context for
// handlers. Non-members and under-privileged members receive 403 — membership
// existence is never revealed to outsiders.
func RequireCapability(svc *Service, cap Capability, log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := web.UserID(r.Context())
			if !ok {
				web.Fail(w, r, log, apperr.Unauthorized("missing_token", "authentication required"))
				return
			}
			workspaceID, err := uuid.Parse(chi.URLParam(r, workspaceURLParam))
			if err != nil {
				web.Fail(w, r, log, apperr.Validation("invalid_workspace_id", "invalid workspace id"))
				return
			}

			member, err := svc.Membership(r.Context(), workspaceID, userID)
			if err != nil {
				// Collapse not-found to forbidden so non-members can't probe existence.
				web.Fail(w, r, log, apperr.Forbidden("forbidden", "you do not have access to this workspace"))
				return
			}
			if !member.Has(cap) {
				web.Fail(w, r, log, apperr.Forbidden("forbidden", "you lack the required capability: "+string(cap)))
				return
			}
			next.ServeHTTP(w, r.WithContext(withMember(r.Context(), member)))
		})
	}
}
