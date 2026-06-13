package workspace

import (
	"log/slog"
	"net"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Akins20/postal/internal/platform/apperr"
	"github.com/Akins20/postal/internal/platform/web"
)

// Handler serves the /api/v1/workspaces endpoints. RequireUser is applied by the
// server when mounting this router; capability checks are applied per route.
type Handler struct {
	svc *Service
	log *slog.Logger
}

// NewHandler builds the workspace HTTP handler.
func NewHandler(svc *Service, log *slog.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

// RegisterTop registers routes at the /workspaces root (mounted behind
// RequireUser by the server).
func (h *Handler) RegisterTop(r chi.Router) {
	r.Get("/", web.Handler(h.log, h.list))
}

// RegisterWorkspaceScoped registers member routes onto a router already scoped
// to /workspaces/{workspaceID}. The server composes this subtree so multiple
// domains (members, channels) can share the single {workspaceID} route.
func (h *Handler) RegisterWorkspaceScoped(r chi.Router) {
	r.With(RequireCapability(h.svc, CapRead, h.log)).
		Get("/members", web.Handler(h.log, h.listMembers))
	r.With(RequireCapability(h.svc, CapManageMembers, h.log)).
		Post("/members", web.Handler(h.log, h.addMember))
	r.With(RequireCapability(h.svc, CapManageMembers, h.log)).
		Patch("/members/{userID}/capabilities", web.Handler(h.log, h.updateCapabilities))
	r.With(RequireCapability(h.svc, CapManageMembers, h.log)).
		Get("/members/{userID}/channels", web.Handler(h.log, h.getMemberChannels))
	r.With(RequireCapability(h.svc, CapManageMembers, h.log)).
		Put("/members/{userID}/channels", web.Handler(h.log, h.setMemberChannels))
	r.With(RequireCapability(h.svc, CapManageMembers, h.log)).
		Get("/activity", web.Handler(h.log, h.activity))
}

type setChannelAccessRequest struct {
	Restricted bool        `json:"restricted"`
	ChannelIDs []uuid.UUID `json:"channel_ids"`
}

// getMemberChannels returns a member's per-channel publish access.
func (h *Handler) getMemberChannels(w http.ResponseWriter, r *http.Request) error {
	workspaceID, err := uuid.Parse(chi.URLParam(r, workspaceURLParam))
	if err != nil {
		return apperr.Validation("invalid_workspace_id", "invalid workspace id")
	}
	targetID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		return apperr.Validation("invalid_user_id", "invalid user id")
	}
	access, err := h.svc.GetMemberChannelAccess(r.Context(), workspaceID, targetID)
	if err != nil {
		return err
	}
	web.Respond(w, http.StatusOK, access)
	return nil
}

// setMemberChannels replaces a member's per-channel publish allowlist.
func (h *Handler) setMemberChannels(w http.ResponseWriter, r *http.Request) error {
	workspaceID, err := uuid.Parse(chi.URLParam(r, workspaceURLParam))
	if err != nil {
		return apperr.Validation("invalid_workspace_id", "invalid workspace id")
	}
	targetID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		return apperr.Validation("invalid_user_id", "invalid user id")
	}
	var req setChannelAccessRequest
	if err := web.DecodeJSON(w, r, &req); err != nil {
		return err
	}
	if err := h.svc.SetMemberChannelAccess(r.Context(), workspaceID, targetID, req.Restricted, req.ChannelIDs); err != nil {
		return err
	}
	access, err := h.svc.GetMemberChannelAccess(r.Context(), workspaceID, targetID)
	if err != nil {
		return err
	}
	web.Respond(w, http.StatusOK, access)
	return nil
}

// activity returns the workspace's recent audit-log entries (who did what).
func (h *Handler) activity(w http.ResponseWriter, r *http.Request) error {
	workspaceID, err := uuid.Parse(chi.URLParam(r, workspaceURLParam))
	if err != nil {
		return apperr.Validation("invalid_workspace_id", "invalid workspace id")
	}
	entries, err := h.svc.ListActivity(r.Context(), workspaceID, 100)
	if err != nil {
		return err
	}
	web.Respond(w, http.StatusOK, entries)
	return nil
}

// WorkspaceURLParam is the chi route parameter naming the workspace, exposed so
// the server can build the shared /workspaces/{workspaceID} subtree.
const WorkspaceURLParam = workspaceURLParam

type memberResponse struct {
	WorkspaceID uuid.UUID `json:"workspace_id"`
	UserID      uuid.UUID `json:"user_id"`
	Role        string    `json:"role"`
	Permissions []string  `json:"permissions"`
}

type updateCapabilitiesRequest struct {
	Role         string   `json:"role"`
	Capabilities []string `json:"capabilities"`
}

type addMemberRequest struct {
	Email        string   `json:"email"`
	Role         string   `json:"role"`
	Capabilities []string `json:"capabilities"`
}

func (h *Handler) addMember(w http.ResponseWriter, r *http.Request) error {
	actorID, ok := web.UserID(r.Context())
	if !ok {
		return apperr.Unauthorized("missing_token", "authentication required")
	}
	workspaceID, err := uuid.Parse(chi.URLParam(r, workspaceURLParam))
	if err != nil {
		return apperr.Validation("invalid_workspace_id", "invalid workspace id")
	}
	var req addMemberRequest
	if err := web.DecodeJSON(w, r, &req); err != nil {
		return err
	}
	member, err := h.svc.AddMember(r.Context(), actorID, workspaceID, req.Email, req.Role, req.Capabilities, clientIP(r))
	if err != nil {
		return err
	}
	web.Respond(w, http.StatusCreated, toMemberResponse(member))
	return nil
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) error {
	userID, ok := web.UserID(r.Context())
	if !ok {
		return apperr.Unauthorized("missing_token", "authentication required")
	}
	workspaces, err := h.svc.ListForUser(r.Context(), userID)
	if err != nil {
		return err
	}
	web.Respond(w, http.StatusOK, workspaces)
	return nil
}

func (h *Handler) listMembers(w http.ResponseWriter, r *http.Request) error {
	workspaceID, err := uuid.Parse(chi.URLParam(r, workspaceURLParam))
	if err != nil {
		return apperr.Validation("invalid_workspace_id", "invalid workspace id")
	}
	members, err := h.svc.ListMembers(r.Context(), workspaceID)
	if err != nil {
		return err
	}
	web.Respond(w, http.StatusOK, toMemberResponses(members))
	return nil
}

func (h *Handler) updateCapabilities(w http.ResponseWriter, r *http.Request) error {
	actorID, ok := web.UserID(r.Context())
	if !ok {
		return apperr.Unauthorized("missing_token", "authentication required")
	}
	workspaceID, err := uuid.Parse(chi.URLParam(r, workspaceURLParam))
	if err != nil {
		return apperr.Validation("invalid_workspace_id", "invalid workspace id")
	}
	targetID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		return apperr.Validation("invalid_user_id", "invalid user id")
	}

	var req updateCapabilitiesRequest
	if err := web.DecodeJSON(w, r, &req); err != nil {
		return err
	}
	member, err := h.svc.UpdateCapabilities(r.Context(), actorID, workspaceID, targetID, req.Role, req.Capabilities, clientIP(r))
	if err != nil {
		return err
	}
	web.Respond(w, http.StatusOK, toMemberResponse(member))
	return nil
}

func toMemberResponse(m Member) memberResponse {
	perms := m.Permissions
	if perms == nil {
		perms = []string{}
	}
	return memberResponse{WorkspaceID: m.WorkspaceID, UserID: m.UserID, Role: m.Role, Permissions: perms}
}

func toMemberResponses(ms []Member) []memberResponse {
	out := make([]memberResponse, len(ms))
	for i, m := range ms {
		out[i] = toMemberResponse(m)
	}
	return out
}

// clientIP extracts the client IP for audit logging.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
