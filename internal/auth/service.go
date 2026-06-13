package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Akins20/postal/internal/platform/apperr"
	"github.com/Akins20/postal/internal/platform/db"
	"github.com/Akins20/postal/internal/platform/db/sqlc"
	"github.com/Akins20/postal/internal/security"
	"github.com/Akins20/postal/internal/workspace"
)

// Token lifetimes for one-time email links.
const (
	emailVerifyTTL   = 24 * time.Hour
	passwordResetTTL = time.Hour
)

// defaultWorkspaceName is the personal workspace auto-created at signup.
const defaultWorkspaceName = "Personal"

// Service orchestrates identity flows over the database, token issuer, session
// store, mailer, and audit log.
type Service struct {
	pool      *db.Pool
	tokens    *TokenIssuer
	sessions  *SessionStore
	mailer    Mailer
	audit     security.Recorder
	clock     func() time.Time
	dummyHash string
}

// NewService builds an auth Service. clock defaults to time.Now. It precomputes
// a dummy password hash so Login can perform equal Argon2id work whether or not
// the account exists, defeating timing-based account enumeration.
func NewService(pool *db.Pool, tokens *TokenIssuer, sessions *SessionStore, mailer Mailer, audit security.Recorder, clock func() time.Time) *Service {
	if clock == nil {
		clock = time.Now
	}
	dummy, _ := HashPassword("timing-equalization-placeholder")
	return &Service{pool: pool, tokens: tokens, sessions: sessions, mailer: mailer, audit: audit, clock: clock, dummyHash: dummy}
}

// LoginResult carries the user and freshly minted tokens after authentication.
type LoginResult struct {
	User         sqlc.User
	AccessToken  string
	RefreshToken string
}

// RefreshResult carries the rotated tokens after a refresh.
type RefreshResult struct {
	UserID       uuid.UUID
	AccessToken  string
	RefreshToken string
}

// Signup creates a user, their personal workspace, and an owner membership
// atomically, then issues an email-verification token. Email is normalized and
// validated; disposable domains are rejected. A duplicate email yields a
// conflict error.
func (s *Service) Signup(ctx context.Context, email, password, ip string) (sqlc.User, error) {
	email = normalizeEmail(email)
	if err := validateEmail(email); err != nil {
		return sqlc.User{}, err
	}
	if isDisposableEmail(email) {
		return sqlc.User{}, apperr.Validation("disposable_email", "disposable email addresses are not allowed").
			WithField("email", "disposable addresses are not allowed")
	}
	if err := validatePassword(password); err != nil {
		return sqlc.User{}, err
	}

	// If the address already exists but was never verified, treat this as a
	// re-send rather than a conflict: only a verified account is "already
	// registered". This also avoids leaking the existence of unverified accounts
	// and never touches the existing account's password.
	if existing, lookupErr := s.pool.Queries().GetUserByEmail(ctx, email); lookupErr == nil {
		if existing.EmailVerified {
			return sqlc.User{}, apperr.Conflict("email_taken", "that email address is already registered")
		}
		if resendErr := s.ResendEmailVerification(ctx, email, ip); resendErr != nil {
			return sqlc.User{}, resendErr
		}
		return existing, nil
	} else if !errors.Is(lookupErr, pgx.ErrNoRows) {
		return sqlc.User{}, apperr.Internal(lookupErr)
	}

	hash, err := HashPassword(password)
	if err != nil {
		return sqlc.User{}, apperr.Internal(err)
	}
	verifyToken, err := newOpaqueToken()
	if err != nil {
		return sqlc.User{}, apperr.Internal(err)
	}

	var user sqlc.User
	err = s.pool.WithTx(ctx, func(q *sqlc.Queries) error {
		u, err := q.CreateUser(ctx, sqlc.CreateUserParams{Email: email, PasswordHash: hash})
		if err != nil {
			return mapCreateUserErr(err)
		}
		ws, err := q.CreateWorkspace(ctx, sqlc.CreateWorkspaceParams{Name: defaultWorkspaceName, OwnerUserID: u.ID})
		if err != nil {
			return fmt.Errorf("creating personal workspace: %w", err)
		}
		if _, err := q.CreateMember(ctx, sqlc.CreateMemberParams{
			WorkspaceID: ws.ID,
			UserID:      u.ID,
			Role:        string(workspace.RoleOwner),
			Permissions: workspace.PresetCapabilities(workspace.RoleOwner),
		}); err != nil {
			return fmt.Errorf("creating owner membership: %w", err)
		}
		if _, err := q.CreateEmailVerificationToken(ctx, sqlc.CreateEmailVerificationTokenParams{
			UserID:    u.ID,
			TokenHash: hashToken(verifyToken),
			ExpiresAt: tsFromTime(s.clock().Add(emailVerifyTTL)),
		}); err != nil {
			return fmt.Errorf("creating verification token: %w", err)
		}
		user = u
		return nil
	})
	if err != nil {
		return sqlc.User{}, err
	}

	// Email delivery and audit are best-effort side effects outside the tx.
	if mailErr := s.mailer.SendEmailVerification(ctx, email, verifyToken); mailErr != nil {
		_ = mailErr // console mailer never errors; real mailer failures are non-fatal to signup
	}
	s.recordAudit(ctx, &user.ID, "user.signup", ip, map[string]any{"email": email})
	return user, nil
}

// Login verifies credentials and issues an access token plus a refresh token.
// Failures return a single generic error to avoid account enumeration.
func (s *Service) Login(ctx context.Context, email, password, ip string) (LoginResult, error) {
	email = normalizeEmail(email)

	user, err := s.pool.Queries().GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Spend the same Argon2id work as a real verify so response time
			// does not reveal whether the account exists (enumeration defense).
			_, _ = VerifyPassword(password, s.dummyHash)
			return LoginResult{}, invalidCredentials()
		}
		return LoginResult{}, apperr.Internal(err)
	}

	ok, err := VerifyPassword(password, user.PasswordHash)
	if err != nil || !ok {
		return LoginResult{}, invalidCredentials()
	}
	if user.Status != "active" {
		return LoginResult{}, apperr.Forbidden("account_inactive", "this account is not active")
	}
	// Email must be verified before a session is issued. The dedicated code lets
	// clients offer a "resend verification" affordance instead of a dead end.
	if !user.EmailVerified {
		return LoginResult{}, apperr.Forbidden("email_not_verified", "please verify your email address before signing in")
	}

	access, err := s.tokens.Issue(user.ID)
	if err != nil {
		return LoginResult{}, apperr.Internal(err)
	}
	refresh, err := s.sessions.Create(ctx, user.ID)
	if err != nil {
		return LoginResult{}, apperr.Internal(err)
	}

	s.recordAudit(ctx, &user.ID, "user.login", ip, nil)
	return LoginResult{User: user, AccessToken: access, RefreshToken: refresh}, nil
}

// Refresh rotates the refresh token (sliding the session) and issues a new
// access token. An invalid/expired refresh token is unauthorized.
func (s *Service) Refresh(ctx context.Context, refreshToken string) (RefreshResult, error) {
	newRefresh, userID, err := s.sessions.Rotate(ctx, refreshToken)
	if errors.Is(err, ErrInvalidSession) {
		return RefreshResult{}, apperr.Unauthorized("invalid_session", "session expired; please log in again")
	}
	if err != nil {
		return RefreshResult{}, apperr.Internal(err)
	}
	access, err := s.tokens.Issue(userID)
	if err != nil {
		return RefreshResult{}, apperr.Internal(err)
	}
	return RefreshResult{UserID: userID, AccessToken: access, RefreshToken: newRefresh}, nil
}

// GetUser loads a user by ID, mapping a missing row to a not-found error.
func (s *Service) GetUser(ctx context.Context, id uuid.UUID) (sqlc.User, error) {
	user, err := s.pool.Queries().GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return sqlc.User{}, apperr.NotFound("user_not_found", "user not found")
		}
		return sqlc.User{}, apperr.Internal(err)
	}
	return user, nil
}

// Logout revokes the refresh token, ending the session.
func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	if err := s.sessions.Revoke(ctx, refreshToken); err != nil {
		return apperr.Internal(err)
	}
	return nil
}

// recordAudit best-effort writes an audit entry; failures are swallowed so they
// never break the primary flow.
func (s *Service) recordAudit(ctx context.Context, actor *uuid.UUID, action, ip string, meta map[string]any) {
	if s.audit == nil {
		return
	}
	_ = s.audit.Record(ctx, security.Event{ActorUserID: actor, Action: action, IP: ip, Metadata: meta})
}

// invalidCredentials returns the generic auth-failure error used for both
// unknown email and wrong password, preventing enumeration.
func invalidCredentials() error {
	return apperr.Unauthorized("invalid_credentials", "invalid email or password")
}

// mapCreateUserErr converts a duplicate-email unique violation into a conflict.
func mapCreateUserErr(err error) error {
	if db.IsUniqueViolation(err) {
		return apperr.Conflict("email_taken", "that email address is already registered")
	}
	return fmt.Errorf("creating user: %w", err)
}

// tsFromTime converts a time.Time to a valid pgtype.Timestamptz.
func tsFromTime(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}
