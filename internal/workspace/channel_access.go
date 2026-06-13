package workspace

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/Akins20/postal/internal/platform/apperr"
	"github.com/Akins20/postal/internal/platform/db/sqlc"
)

// ChannelAccess is a member's per-channel publish access. Restricted=false (the
// default) grants every channel; when true, only AllowedChannelIDs may be used.
type ChannelAccess struct {
	Restricted        bool        `json:"restricted"`
	AllowedChannelIDs []uuid.UUID `json:"allowed_channel_ids"`
}

// GetMemberChannelAccess returns a member's per-channel publish access.
func (s *Service) GetMemberChannelAccess(ctx context.Context, workspaceID, userID uuid.UUID) (ChannelAccess, error) {
	restricted, err := s.pool.Queries().GetMemberChannelRestricted(ctx,
		sqlc.GetMemberChannelRestrictedParams{WorkspaceID: workspaceID, UserID: userID})
	if err != nil {
		return ChannelAccess{}, apperr.Internal(err)
	}
	ids, err := s.pool.Queries().ListChannelGrantsForUser(ctx,
		sqlc.ListChannelGrantsForUserParams{WorkspaceID: workspaceID, UserID: userID})
	if err != nil {
		return ChannelAccess{}, apperr.Internal(err)
	}
	return ChannelAccess{Restricted: restricted, AllowedChannelIDs: ids}, nil
}

// SetMemberChannelAccess replaces a member's per-channel publish allowlist. When
// restricted is false the grant list is irrelevant (full access); we still clear
// it so re-enabling restriction starts from a clean slate.
func (s *Service) SetMemberChannelAccess(ctx context.Context, workspaceID, userID uuid.UUID, restricted bool, channelIDs []uuid.UUID) error {
	err := s.pool.WithTx(ctx, func(q *sqlc.Queries) error {
		if err := q.SetMemberChannelRestricted(ctx, sqlc.SetMemberChannelRestrictedParams{
			WorkspaceID: workspaceID, UserID: userID, ChannelRestricted: restricted,
		}); err != nil {
			return err
		}
		if err := q.DeleteChannelGrantsForUser(ctx, sqlc.DeleteChannelGrantsForUserParams{
			WorkspaceID: workspaceID, UserID: userID,
		}); err != nil {
			return err
		}
		if restricted {
			for _, cid := range channelIDs {
				if err := q.InsertChannelGrant(ctx, sqlc.InsertChannelGrantParams{
					WorkspaceID: workspaceID, ChannelID: cid, UserID: userID,
				}); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return apperr.Internal(err)
	}
	return nil
}

// CanPublishToChannel reports whether userID may publish to channelID. An
// unrestricted member, or a channel on a restricted member's allowlist, is
// allowed. A nil result (no membership row) is denied.
func (s *Service) CanPublishToChannel(ctx context.Context, workspaceID, userID, channelID uuid.UUID) (bool, error) {
	allowed, err := s.pool.Queries().IsChannelPublishAllowed(ctx, sqlc.IsChannelPublishAllowedParams{
		ChannelID: channelID, WorkspaceID: workspaceID, UserID: userID,
	})
	if err != nil {
		return false, err
	}
	return allowed != nil && *allowed, nil
}

// ActivityEntry is one audit-log line for the "who did what" activity feed.
type ActivityEntry struct {
	ID         int64     `json:"id"`
	ActorEmail string    `json:"actor_email"`
	Action     string    `json:"action"`
	Target     string    `json:"target"`
	CreatedAt  time.Time `json:"created_at"`
}

// ListActivity returns the most recent workspace audit entries (newest first).
func (s *Service) ListActivity(ctx context.Context, workspaceID uuid.UUID, limit int32) ([]ActivityEntry, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	rows, err := s.pool.Queries().ListWorkspaceActivity(ctx, sqlc.ListWorkspaceActivityParams{
		WorkspaceID: &workspaceID, Limit: limit,
	})
	if err != nil {
		return nil, apperr.Internal(err)
	}
	out := make([]ActivityEntry, 0, len(rows))
	for _, r := range rows {
		email := ""
		if r.ActorEmail != nil {
			email = *r.ActorEmail
		}
		out = append(out, ActivityEntry{
			ID: r.ID, ActorEmail: email, Action: r.Action, Target: r.Target, CreatedAt: r.CreatedAt.Time,
		})
	}
	return out, nil
}
