package analytics

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Akins20/postal/internal/platform/apperr"
	"github.com/Akins20/postal/internal/platform/web"
	"github.com/Akins20/postal/internal/workspace"
)

// defaultSeriesWindow bounds an unbounded series query.
const defaultSeriesWindow = 30 * 24 * time.Hour

// Handler serves analytics reporting endpoints under /workspaces/{workspaceID}.
// All reads require the read capability and are workspace-isolated.
type Handler struct {
	svc   *Service
	wsSvc *workspace.Service
	log   *slog.Logger
}

// NewHandler builds the analytics HTTP handler.
func NewHandler(svc *Service, wsSvc *workspace.Service, log *slog.Logger) *Handler {
	return &Handler{svc: svc, wsSvc: wsSvc, log: log}
}

// RegisterWorkspaceScoped registers analytics routes onto a /workspaces/{workspaceID} router.
func (h *Handler) RegisterWorkspaceScoped(r chi.Router) {
	r.Route("/analytics", func(ar chi.Router) {
		ar.Use(workspace.RequireCapability(h.wsSvc, workspace.CapRead, h.log))
		ar.Get("/", web.Handler(h.log, h.overview))
		ar.Get("/export.csv", web.Handler(h.log, h.export))
		ar.Get("/posts/{postID}", web.Handler(h.log, h.post))
		ar.Get("/posts/{postID}/series", web.Handler(h.log, h.series))
	})
}

func (h *Handler) overview(w http.ResponseWriter, r *http.Request) error {
	wsID, err := web.PathUUID(r, workspace.WorkspaceURLParam)
	if err != nil {
		return err
	}
	posts, err := h.svc.WorkspaceOverview(r.Context(), wsID)
	if err != nil {
		return err
	}
	web.Respond(w, http.StatusOK, map[string]any{"posts": posts})
	return nil
}

func (h *Handler) post(w http.ResponseWriter, r *http.Request) error {
	wsID, err := web.PathUUID(r, workspace.WorkspaceURLParam)
	if err != nil {
		return err
	}
	postID, err := web.PathUUID(r, "postID")
	if err != nil {
		return err
	}
	channels, err := h.svc.LatestForPost(r.Context(), wsID, postID)
	if err != nil {
		return err
	}
	web.Respond(w, http.StatusOK, map[string]any{"post_id": postID, "channels": channels})
	return nil
}

func (h *Handler) series(w http.ResponseWriter, r *http.Request) error {
	wsID, err := web.PathUUID(r, workspace.WorkspaceURLParam)
	if err != nil {
		return err
	}
	postID, err := web.PathUUID(r, "postID")
	if err != nil {
		return err
	}
	channelID, err := uuid.Parse(r.URL.Query().Get("channel_id"))
	if err != nil {
		return apperr.Validation("invalid_channel_id", "channel_id query parameter is required")
	}
	now := time.Now().UTC()
	from, to, err := web.TimeRange(r, now.Add(-defaultSeriesWindow), now)
	if err != nil {
		return err
	}
	metric := r.URL.Query().Get("metric")
	points, err := h.svc.Series(r.Context(), wsID, postID, channelID, metric, from, to)
	if err != nil {
		return err
	}
	web.Respond(w, http.StatusOK, map[string]any{"post_id": postID, "channel_id": channelID, "metric": metric, "points": points})
	return nil
}

func (h *Handler) export(w http.ResponseWriter, r *http.Request) error {
	wsID, err := web.PathUUID(r, workspace.WorkspaceURLParam)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="analytics.csv"`)
	w.WriteHeader(http.StatusOK)
	// Headers are sent; a write error mid-stream can't be reported to the client.
	_ = h.svc.ExportCSV(r.Context(), wsID, w)
	return nil
}
