package integration

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Akins20/postal/internal/platform/apperr"
	"github.com/Akins20/postal/internal/platform/web"
	"github.com/Akins20/postal/internal/workspace"
)

// Handler serves /workspaces/{workspaceID}/integrations. Reading needs read;
// configuring needs manage_workspace; the shorten action needs create (it is
// compose assistance).
type Handler struct {
	svc   *Service
	wsSvc *workspace.Service
	log   *slog.Logger
}

// NewHandler builds the integrations HTTP handler.
func NewHandler(svc *Service, wsSvc *workspace.Service, log *slog.Logger) *Handler {
	return &Handler{svc: svc, wsSvc: wsSvc, log: log}
}

// RegisterWorkspaceScoped registers integration routes onto a
// /workspaces/{workspaceID} router.
func (h *Handler) RegisterWorkspaceScoped(r chi.Router) {
	r.Route("/integrations", func(ir chi.Router) {
		ir.With(workspace.RequireCapability(h.wsSvc, workspace.CapRead, h.log)).Get("/", web.Handler(h.log, h.list))
		ir.With(workspace.RequireCapability(h.wsSvc, workspace.CapManageWorkspace, h.log)).Put("/{provider}", web.Handler(h.log, h.configure))
		ir.With(workspace.RequireCapability(h.wsSvc, workspace.CapCreate, h.log)).Post("/ogshortener/shorten", web.Handler(h.log, h.shorten))
	})
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) error {
	wsID, err := web.PathUUID(r, workspace.WorkspaceURLParam)
	if err != nil {
		return err
	}
	items, err := h.svc.List(r.Context(), wsID)
	if err != nil {
		return err
	}
	web.Respond(w, http.StatusOK, items)
	return nil
}

func (h *Handler) configure(w http.ResponseWriter, r *http.Request) error {
	wsID, err := web.PathUUID(r, workspace.WorkspaceURLParam)
	if err != nil {
		return err
	}
	var in struct {
		Enabled   bool    `json:"enabled"`
		AutoApply bool    `json:"auto_apply"`
		APIKey    *string `json:"api_key"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 8192)).Decode(&in); err != nil {
		return apperr.Validation("invalid_json", "could not parse request body")
	}
	it, err := h.svc.Configure(r.Context(), wsID, chi.URLParam(r, "provider"), in.Enabled, in.AutoApply, in.APIKey)
	switch {
	case errors.Is(err, ErrBadKey):
		return apperr.Validation("invalid_api_key", "the provider rejected that API key")
	case errors.Is(err, ErrNotConfigured):
		return apperr.Validation("key_required", "add the provider API key before enabling")
	case err != nil:
		return err
	}
	web.Respond(w, http.StatusOK, it)
	return nil
}

func (h *Handler) shorten(w http.ResponseWriter, r *http.Request) error {
	wsID, err := web.PathUUID(r, workspace.WorkspaceURLParam)
	if err != nil {
		return err
	}
	var in struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 64<<10)).Decode(&in); err != nil {
		return apperr.Validation("invalid_json", "could not parse request body")
	}
	if in.Text == "" {
		return apperr.Validation("missing_text", "expected text to shorten")
	}
	out, err := h.svc.ShortenText(r.Context(), wsID, in.Text)
	if errors.Is(err, ErrNotConfigured) {
		return apperr.Validation("integration_not_configured",
			"enable OGShortener with your API key on the Integrations page first")
	}
	if err != nil {
		return apperr.Validation("shorten_failed", "could not shorten the links in this text")
	}
	web.Respond(w, http.StatusOK, map[string]string{"text": out})
	return nil
}
