package schedule

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

// defaultCalendarWindow bounds an unbounded calendar query.
const defaultCalendarWindow = 30 * 24 * time.Hour

// Handler serves scheduling endpoints under /workspaces/{workspaceID}. Schedule,
// cancel, and slot mutations require the publish capability; reads require read.
type Handler struct {
	svc   *Service
	wsSvc *workspace.Service
	log   *slog.Logger
}

// NewHandler builds the schedule HTTP handler.
func NewHandler(svc *Service, wsSvc *workspace.Service, log *slog.Logger) *Handler {
	return &Handler{svc: svc, wsSvc: wsSvc, log: log}
}

// RegisterWorkspaceScoped registers schedule routes onto a /workspaces/{workspaceID} router.
func (h *Handler) RegisterWorkspaceScoped(r chi.Router) {
	r.With(workspace.RequireCapability(h.wsSvc, workspace.CapPublish, h.log)).Post("/schedule", web.Handler(h.log, h.schedule))
	r.With(workspace.RequireCapability(h.wsSvc, workspace.CapRead, h.log)).Get("/calendar", web.Handler(h.log, h.calendar))
	r.With(workspace.RequireCapability(h.wsSvc, workspace.CapPublish, h.log)).Delete("/scheduled-jobs/{jobID}", web.Handler(h.log, h.cancel))
	r.Route("/slots", func(sr chi.Router) {
		sr.With(workspace.RequireCapability(h.wsSvc, workspace.CapRead, h.log)).Get("/", web.Handler(h.log, h.listSlots))
		sr.With(workspace.RequireCapability(h.wsSvc, workspace.CapPublish, h.log)).Post("/", web.Handler(h.log, h.createSlot))
		sr.With(workspace.RequireCapability(h.wsSvc, workspace.CapPublish, h.log)).Delete("/{slotID}", web.Handler(h.log, h.deleteSlot))
	})
}

type scheduleRequest struct {
	PostID  uuid.UUID `json:"post_id"`
	RunAt   time.Time `json:"run_at"`
	ToSlots bool      `json:"to_slots"`
}

type slotRequest struct {
	ChannelID uuid.UUID `json:"channel_id"`
	DayOfWeek int       `json:"day_of_week"`
	TimeOfDay string    `json:"time_of_day"`
	Timezone  string    `json:"timezone"`
}

func (h *Handler) schedule(w http.ResponseWriter, r *http.Request) error {
	wsID, err := web.PathUUID(r, workspace.WorkspaceURLParam)
	if err != nil {
		return err
	}
	var req scheduleRequest
	if err := web.DecodeJSON(w, r, &req); err != nil {
		return err
	}
	if req.PostID == uuid.Nil {
		return apperr.Validation("missing_post_id", "post_id is required")
	}

	var jobs []Job
	if req.ToSlots {
		jobs, err = h.svc.ScheduleToSlots(r.Context(), wsID, req.PostID)
	} else {
		if req.RunAt.IsZero() {
			return apperr.Validation("missing_run_at", "run_at is required unless to_slots is set")
		}
		jobs, err = h.svc.SchedulePost(r.Context(), wsID, req.PostID, req.RunAt)
	}
	if err != nil {
		return err
	}
	web.Respond(w, http.StatusCreated, map[string]any{"jobs": jobs})
	return nil
}

func (h *Handler) calendar(w http.ResponseWriter, r *http.Request) error {
	wsID, err := web.PathUUID(r, workspace.WorkspaceURLParam)
	if err != nil {
		return err
	}
	from, to, err := calendarRange(r)
	if err != nil {
		return err
	}
	jobs, err := h.svc.Calendar(r.Context(), wsID, from, to)
	if err != nil {
		return err
	}
	web.Respond(w, http.StatusOK, map[string]any{"jobs": jobs})
	return nil
}

func (h *Handler) cancel(w http.ResponseWriter, r *http.Request) error {
	wsID, err := web.PathUUID(r, workspace.WorkspaceURLParam)
	if err != nil {
		return err
	}
	jobID, err := web.PathUUID(r, "jobID")
	if err != nil {
		return err
	}
	if err := h.svc.Cancel(r.Context(), wsID, jobID); err != nil {
		return err
	}
	web.Respond(w, http.StatusOK, map[string]string{"message": "scheduled job canceled"})
	return nil
}

func (h *Handler) createSlot(w http.ResponseWriter, r *http.Request) error {
	wsID, err := web.PathUUID(r, workspace.WorkspaceURLParam)
	if err != nil {
		return err
	}
	var req slotRequest
	if err := web.DecodeJSON(w, r, &req); err != nil {
		return err
	}
	slot, err := h.svc.CreateSlot(r.Context(), wsID, req.ChannelID, req.DayOfWeek, req.TimeOfDay, req.Timezone)
	if err != nil {
		return err
	}
	web.Respond(w, http.StatusCreated, slot)
	return nil
}

func (h *Handler) listSlots(w http.ResponseWriter, r *http.Request) error {
	wsID, err := web.PathUUID(r, workspace.WorkspaceURLParam)
	if err != nil {
		return err
	}
	channelID, err := uuid.Parse(r.URL.Query().Get("channel_id"))
	if err != nil {
		return apperr.Validation("invalid_channel_id", "channel_id query parameter is required")
	}
	slots, err := h.svc.ListSlots(r.Context(), wsID, channelID)
	if err != nil {
		return err
	}
	web.Respond(w, http.StatusOK, slots)
	return nil
}

func (h *Handler) deleteSlot(w http.ResponseWriter, r *http.Request) error {
	wsID, err := web.PathUUID(r, workspace.WorkspaceURLParam)
	if err != nil {
		return err
	}
	slotID, err := web.PathUUID(r, "slotID")
	if err != nil {
		return err
	}
	channelID, err := uuid.Parse(r.URL.Query().Get("channel_id"))
	if err != nil {
		return apperr.Validation("invalid_channel_id", "channel_id query parameter is required")
	}
	if err := h.svc.DeleteSlot(r.Context(), wsID, channelID, slotID); err != nil {
		return err
	}
	web.Respond(w, http.StatusOK, map[string]string{"message": "slot deleted"})
	return nil
}

// calendarRange parses ?from=&to= (RFC3339), defaulting to [now, now+30d).
func calendarRange(r *http.Request) (from, to time.Time, err error) {
	now := time.Now().UTC()
	return web.TimeRange(r, now, now.Add(defaultCalendarWindow))
}
