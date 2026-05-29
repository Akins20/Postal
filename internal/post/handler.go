package post

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Akins20/postal/internal/platform/web"
	"github.com/Akins20/postal/internal/workspace"
)

// Handler serves the composer endpoints under /workspaces/{workspaceID}/posts.
// Capability gating: read to list/get/validate, create/update/delete for writes.
type Handler struct {
	svc   *Service
	wsSvc *workspace.Service
	log   *slog.Logger
}

// NewHandler builds the post HTTP handler. wsSvc backs capability checks.
func NewHandler(svc *Service, wsSvc *workspace.Service, log *slog.Logger) *Handler {
	return &Handler{svc: svc, wsSvc: wsSvc, log: log}
}

// RegisterWorkspaceScoped registers post routes onto a router scoped to
// /workspaces/{workspaceID}.
func (h *Handler) RegisterWorkspaceScoped(r chi.Router) {
	r.Route("/posts", func(pr chi.Router) {
		pr.With(workspace.RequireCapability(h.wsSvc, workspace.CapRead, h.log)).Get("/", web.Handler(h.log, h.list))
		pr.With(workspace.RequireCapability(h.wsSvc, workspace.CapCreate, h.log)).Post("/", web.Handler(h.log, h.create))
		pr.With(workspace.RequireCapability(h.wsSvc, workspace.CapRead, h.log)).Post("/utm-preview", web.Handler(h.log, h.utmPreview))
		pr.With(workspace.RequireCapability(h.wsSvc, workspace.CapRead, h.log)).Get("/{postID}", web.Handler(h.log, h.get))
		pr.With(workspace.RequireCapability(h.wsSvc, workspace.CapUpdate, h.log)).Put("/{postID}", web.Handler(h.log, h.update))
		pr.With(workspace.RequireCapability(h.wsSvc, workspace.CapDelete, h.log)).Delete("/{postID}", web.Handler(h.log, h.delete))
		pr.With(workspace.RequireCapability(h.wsSvc, workspace.CapRead, h.log)).Post("/{postID}/validate", web.Handler(h.log, h.validate))
	})
}

type variantsRequest struct {
	Variants []VariantInput `json:"variants"`
}

type utmRequest struct {
	Text string            `json:"text"`
	UTM  map[string]string `json:"utm"`
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) error {
	wsID, err := workspaceID(r)
	if err != nil {
		return err
	}
	posts, err := h.svc.List(r.Context(), wsID)
	if err != nil {
		return err
	}
	web.Respond(w, http.StatusOK, posts)
	return nil
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) error {
	wsID, err := workspaceID(r)
	if err != nil {
		return err
	}
	userID, _ := web.UserID(r.Context())
	var req variantsRequest
	if err := web.DecodeJSON(w, r, &req); err != nil {
		return err
	}
	p, err := h.svc.Create(r.Context(), wsID, userID, req.Variants)
	if err != nil {
		return err
	}
	web.Respond(w, http.StatusCreated, p)
	return nil
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) error {
	wsID, err := workspaceID(r)
	if err != nil {
		return err
	}
	postID, err := web.PathUUID(r, "postID")
	if err != nil {
		return err
	}
	p, err := h.svc.Get(r.Context(), wsID, postID)
	if err != nil {
		return err
	}
	web.Respond(w, http.StatusOK, p)
	return nil
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) error {
	wsID, err := workspaceID(r)
	if err != nil {
		return err
	}
	postID, err := web.PathUUID(r, "postID")
	if err != nil {
		return err
	}
	userID, _ := web.UserID(r.Context())
	var req variantsRequest
	if err := web.DecodeJSON(w, r, &req); err != nil {
		return err
	}
	p, err := h.svc.Update(r.Context(), wsID, userID, postID, req.Variants)
	if err != nil {
		return err
	}
	web.Respond(w, http.StatusOK, p)
	return nil
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) error {
	wsID, err := workspaceID(r)
	if err != nil {
		return err
	}
	postID, err := web.PathUUID(r, "postID")
	if err != nil {
		return err
	}
	userID, _ := web.UserID(r.Context())
	if err := h.svc.Delete(r.Context(), wsID, userID, postID); err != nil {
		return err
	}
	web.Respond(w, http.StatusOK, map[string]string{"message": "post deleted"})
	return nil
}

func (h *Handler) validate(w http.ResponseWriter, r *http.Request) error {
	wsID, err := workspaceID(r)
	if err != nil {
		return err
	}
	postID, err := web.PathUUID(r, "postID")
	if err != nil {
		return err
	}
	results, err := h.svc.Validate(r.Context(), wsID, postID)
	if err != nil {
		return err
	}
	web.Respond(w, http.StatusOK, map[string]any{"variants": results})
	return nil
}

func (h *Handler) utmPreview(w http.ResponseWriter, r *http.Request) error {
	var req utmRequest
	if err := web.DecodeJSON(w, r, &req); err != nil {
		return err
	}
	web.Respond(w, http.StatusOK, map[string]string{"text": ApplyUTM(req.Text, req.UTM)})
	return nil
}

// workspaceID parses the {workspaceID} route param.
func workspaceID(r *http.Request) (uuid.UUID, error) {
	return web.PathUUID(r, workspace.WorkspaceURLParam)
}
