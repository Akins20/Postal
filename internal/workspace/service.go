package workspace

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Akins20/postal/internal/platform/apperr"
	"github.com/Akins20/postal/internal/platform/db"
	"github.com/Akins20/postal/internal/platform/db/sqlc"
	"github.com/Akins20/postal/internal/security"
)

// Service implements workspace queries and capability management.
type Service struct {
	pool  *db.Pool
	audit security.Recorder
	clock func() time.Time
}

// NewService builds a workspace Service. clock defaults to time.Now.
func NewService(pool *db.Pool, audit security.Recorder, clock func() time.Time) *Service {
	if clock == nil {
		clock = time.Now
	}
	return &Service{pool: pool, audit: audit, clock: clock}
}

// Workspace is the API representation of a workspace.
type Workspace struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	OwnerUserID uuid.UUID `json:"owner_user_id"`
	Plan        string    `json:"plan"`
	CreatedAt   time.Time `json:"created_at"`
}

// ListForUser returns the workspaces the user belongs to.
func (s *Service) ListForUser(ctx context.Context, userID uuid.UUID) ([]Workspace, error) {
	rows, err := s.pool.Queries().ListWorkspacesForUser(ctx, userID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	out := make([]Workspace, len(rows))
	for i, r := range rows {
		out[i] = Workspace{
			ID:          r.ID,
			Name:        r.Name,
			OwnerUserID: r.OwnerUserID,
			Plan:        r.Plan,
			CreatedAt:   r.CreatedAt.Time,
		}
	}
	return out, nil
}

// Membership returns a user's membership in a workspace, or a not-found error if
// the user is not a member.
func (s *Service) Membership(ctx context.Context, workspaceID, userID uuid.UUID) (Member, error) {
	row, err := s.pool.Queries().GetMember(ctx, sqlc.GetMemberParams{WorkspaceID: workspaceID, UserID: userID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Member{}, apperr.NotFound("not_a_member", "not a member of this workspace")
		}
		return Member{}, apperr.Internal(err)
	}
	return toMember(row), nil
}

// ListMembers returns all members of a workspace.
func (s *Service) ListMembers(ctx context.Context, workspaceID uuid.UUID) ([]Member, error) {
	rows, err := s.pool.Queries().ListMembers(ctx, workspaceID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	out := make([]Member, len(rows))
	for i, r := range rows {
		out[i] = toMember(r)
	}
	return out, nil
}

// AddMember adds an existing user (looked up by email) to the workspace with the
// given role/capabilities. The actor must be able to grant every capability
// (no privilege escalation). A user who is already a member yields a conflict.
func (s *Service) AddMember(ctx context.Context, actorID, workspaceID uuid.UUID, email, role string, caps []string, ip string) (Member, error) {
	actor, err := s.Membership(ctx, workspaceID, actorID)
	if err != nil {
		return Member{}, err
	}

	user, err := s.pool.Queries().GetUserByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Member{}, apperr.NotFound("user_not_found", "no user with that email")
		}
		return Member{}, apperr.Internal(err)
	}

	finalRole, finalCaps, err := resolveCapabilities(role, caps)
	if err != nil {
		return Member{}, err
	}
	if ok, missing := CanGrant(actor.Permissions, finalCaps); !ok {
		return Member{}, apperr.Forbidden("privilege_escalation", "you cannot grant a capability you do not hold: "+missing)
	}

	row, err := s.pool.Queries().CreateMember(ctx, sqlc.CreateMemberParams{
		WorkspaceID: workspaceID,
		UserID:      user.ID,
		Role:        finalRole,
		Permissions: finalCaps,
	})
	if err != nil {
		if db.IsUniqueViolation(err) {
			return Member{}, apperr.Conflict("already_member", "that user is already a member")
		}
		return Member{}, apperr.Internal(err)
	}

	s.recordAudit(ctx, actorID, workspaceID, user.ID, finalCaps, ip)
	return toMember(row), nil
}

// UpdateCapabilities sets a target member's role and capability set, enforcing
// the authorization invariants: the actor must be able to grant every requested
// capability (no privilege escalation), and the workspace owner's membership is
// immutable. Either an explicit capability list or a role preset must be given.
func (s *Service) UpdateCapabilities(ctx context.Context, actorID, workspaceID, targetID uuid.UUID, role string, caps []string, ip string) (Member, error) {
	actor, err := s.Membership(ctx, workspaceID, actorID)
	if err != nil {
		return Member{}, err
	}
	target, err := s.Membership(ctx, workspaceID, targetID)
	if err != nil {
		return Member{}, err
	}
	if target.Role == string(RoleOwner) {
		return Member{}, apperr.Forbidden("owner_immutable", "the workspace owner's capabilities cannot be changed")
	}

	finalRole, finalCaps, err := resolveCapabilities(role, caps)
	if err != nil {
		return Member{}, err
	}
	if ok, missing := CanGrant(actor.Permissions, finalCaps); !ok {
		return Member{}, apperr.Forbidden("privilege_escalation",
			"you cannot grant a capability you do not hold: "+missing)
	}

	row, err := s.pool.Queries().UpdateMemberPermissions(ctx, sqlc.UpdateMemberPermissionsParams{
		WorkspaceID: workspaceID,
		UserID:      targetID,
		Role:        finalRole,
		Permissions: finalCaps,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Member{}, apperr.NotFound("not_a_member", "not a member of this workspace")
		}
		return Member{}, apperr.Internal(err)
	}

	s.recordAudit(ctx, actorID, workspaceID, targetID, finalCaps, ip)
	return toMember(row), nil
}

// resolveCapabilities determines the final role label and capability set from
// the request: an explicit capability list wins; otherwise a role preset is
// expanded. The role label defaults to "custom" when only capabilities are set,
// signaling the set diverges from any preset.
func resolveCapabilities(role string, caps []string) (string, []string, error) {
	if len(caps) > 0 {
		clean, unknown := NormalizeCapabilities(caps)
		if unknown != "" {
			return "", nil, apperr.Validation("invalid_capability", "unknown capability: "+unknown).
				WithField("capabilities", "unknown capability: "+unknown)
		}
		label := "custom"
		if role != "" {
			label = role
		}
		return label, clean, nil
	}
	if role != "" {
		if !ValidRole(Role(role)) {
			return "", nil, apperr.Validation("invalid_role", "unknown role: "+role).
				WithField("role", "unknown role")
		}
		return role, PresetCapabilities(Role(role)), nil
	}
	return "", nil, apperr.Validation("nothing_to_update", "provide a role or a capabilities list")
}

// recordAudit best-effort logs a capability change.
func (s *Service) recordAudit(ctx context.Context, actorID, workspaceID, targetID uuid.UUID, caps []string, ip string) {
	if s.audit == nil {
		return
	}
	ws := workspaceID
	_ = s.audit.Record(ctx, security.Event{
		WorkspaceID: &ws,
		ActorUserID: &actorID,
		Action:      "member.capabilities_updated",
		Target:      targetID.String(),
		Metadata:    map[string]any{"capabilities": caps},
		IP:          ip,
	})
}

// toMember maps a stored membership row to the domain Member.
func toMember(row sqlc.WorkspaceMember) Member {
	return Member{
		WorkspaceID: row.WorkspaceID,
		UserID:      row.UserID,
		Role:        row.Role,
		Permissions: row.Permissions,
	}
}
