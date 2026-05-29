package channel

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Akins20/postal/internal/platform/apperr"
	"github.com/Akins20/postal/internal/platform/web"
	"github.com/Akins20/postal/internal/workspace"
)

// Handler serves the channel endpoints. Connect/disconnect require the
// manage_channels capability; listing requires read. The OAuth callback is
// authenticated (RequireUser) but not workspace-path-scoped, so it performs its
// own capability re-check in the service.
type Handler struct {
	svc   *Service
	wsSvc *workspace.Service
	log   *slog.Logger
}

// NewHandler builds the channel HTTP handler. wsSvc backs the capability checks.
func NewHandler(svc *Service, wsSvc *workspace.Service, log *slog.Logger) *Handler {
	return &Handler{svc: svc, wsSvc: wsSvc, log: log}
}

// RegisterWorkspaceScoped registers channel routes onto a router scoped to
// /workspaces/{workspaceID}.
func (h *Handler) RegisterWorkspaceScoped(r chi.Router) {
	r.Route("/channels", func(cr chi.Router) {
		cr.With(workspace.RequireCapability(h.wsSvc, workspace.CapRead, h.log)).
			Get("/", web.Handler(h.log, h.list))
		cr.With(workspace.RequireCapability(h.wsSvc, workspace.CapManageChannels, h.log)).
			Post("/connect", web.Handler(h.log, h.connect))
		cr.With(workspace.RequireCapability(h.wsSvc, workspace.CapManageChannels, h.log)).
			Delete("/{channelID}", web.Handler(h.log, h.disconnect))
	})
}

// RegisterCallback registers the OAuth callback at the /api/v1 root (behind
// RequireUser). It is not workspace-scoped; the bound state carries the context.
func (h *Handler) RegisterCallback(r chi.Router) {
	r.Get("/channels/oauth/callback", web.Handler(h.log, h.callback))
}

type connectRequest struct {
	Platform string `json:"platform"`
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) error {
	workspaceID, err := uuid.Parse(chi.URLParam(r, workspace.WorkspaceURLParam))
	if err != nil {
		return apperr.Validation("invalid_workspace_id", "invalid workspace id")
	}
	channels, err := h.svc.ListChannels(r.Context(), workspaceID)
	if err != nil {
		return err
	}
	web.Respond(w, http.StatusOK, channels)
	return nil
}

func (h *Handler) connect(w http.ResponseWriter, r *http.Request) error {
	userID, ok := web.UserID(r.Context())
	if !ok {
		return apperr.Unauthorized("missing_token", "authentication required")
	}
	workspaceID, err := uuid.Parse(chi.URLParam(r, workspace.WorkspaceURLParam))
	if err != nil {
		return apperr.Validation("invalid_workspace_id", "invalid workspace id")
	}
	var req connectRequest
	if err := web.DecodeJSON(w, r, &req); err != nil {
		return err
	}
	authURL, err := h.svc.StartConnect(r.Context(), workspaceID, userID, req.Platform)
	if err != nil {
		return err
	}
	web.Respond(w, http.StatusOK, map[string]string{"authorize_url": authURL})
	return nil
}

func (h *Handler) disconnect(w http.ResponseWriter, r *http.Request) error {
	userID, ok := web.UserID(r.Context())
	if !ok {
		return apperr.Unauthorized("missing_token", "authentication required")
	}
	workspaceID, err := uuid.Parse(chi.URLParam(r, workspace.WorkspaceURLParam))
	if err != nil {
		return apperr.Validation("invalid_workspace_id", "invalid workspace id")
	}
	channelID, err := uuid.Parse(chi.URLParam(r, "channelID"))
	if err != nil {
		return apperr.Validation("invalid_channel_id", "invalid channel id")
	}
	if err := h.svc.Disconnect(r.Context(), userID, workspaceID, channelID); err != nil {
		return err
	}
	web.Respond(w, http.StatusOK, map[string]string{"message": "channel disconnected"})
	return nil
}

func (h *Handler) callback(w http.ResponseWriter, r *http.Request) error {
	userID, ok := web.UserID(r.Context())
	if !ok {
		return apperr.Unauthorized("missing_token", "authentication required")
	}
	state := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")
	if state == "" || code == "" {
		return apperr.Validation("missing_oauth_params", "missing state or code")
	}
	view, err := h.svc.CompleteConnect(r.Context(), userID, state, code)
	if err != nil {
		return err
	}
	web.Respond(w, http.StatusCreated, view)
	return nil
}
