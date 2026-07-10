package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// Invitation status constants (Phase 15.6).
const (
	InviteStatusPending  = "pending"
	InviteStatusAccepted = "accepted"
	InviteStatusRevoked  = "revoked"
	InviteStatusExpired  = "expired"
)

// InviteTTL is how long a fresh invitation stays valid.
const InviteTTL = 7 * 24 * time.Hour

// Invitation is a pending offer of workspace membership sent to an email address.
type Invitation struct {
	ID          int64
	WorkspaceID int64
	Email       string
	Role        string
	Token       string
	InvitedBy   *int64
	Status      string
	ExpiresAt   time.Time
	CreatedAt   time.Time
	AcceptedAt  *time.Time
	// Joined for display.
	WorkspaceName  string
	InvitedByEmail string
}

const invitationColumns = `id, workspace_id, email, role, token, invited_by, status, expires_at, created_at, accepted_at`

func scanInvitation(row pgx.Row) (*Invitation, error) {
	inv := &Invitation{}
	err := row.Scan(&inv.ID, &inv.WorkspaceID, &inv.Email, &inv.Role, &inv.Token,
		&inv.InvitedBy, &inv.Status, &inv.ExpiresAt, &inv.CreatedAt, &inv.AcceptedAt)
	if err != nil {
		return nil, err
	}
	return inv, nil
}

// CreateInvitation creates a pending invite. Only non-owner roles may be invited;
// ownership is granted via TransferOwnership. If a pending invite already exists
// for (workspace, email) it is replaced with a fresh token and expiry (idempotent
// invite / implicit resend).
func (s *AuthStore) CreateInvitation(ctx context.Context, workspaceID int64, email, role string, invitedBy int64) (*Invitation, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return nil, errors.New("email is required")
	}
	if role == "" {
		role = RoleMember
	}
	if role == RoleOwner || !ValidWorkspaceRole(role) {
		return nil, fmt.Errorf("invalid invite role %q", role)
	}

	// Reject if the invitee is already a member.
	var alreadyMember bool
	if err := s.Pool.QueryRow(ctx,
		`SELECT EXISTS(
		   SELECT 1 FROM memberships m JOIN users u ON u.id = m.user_id
		   WHERE m.workspace_id = $1 AND lower(u.email) = $2)`,
		workspaceID, email).Scan(&alreadyMember); err != nil {
		return nil, fmt.Errorf("check existing member: %w", err)
	}
	if alreadyMember {
		return nil, errors.New("that user is already a member of this workspace")
	}

	token, err := GenerateSessionID() // 32 random bytes, hex — reused as invite token
	if err != nil {
		return nil, err
	}
	expiresAt := time.Now().Add(InviteTTL)

	// Supersede any existing pending invite for this (workspace, email).
	if _, err := s.Pool.Exec(ctx,
		`UPDATE invitations SET status = 'revoked'
		 WHERE workspace_id = $1 AND lower(email) = $2 AND status = 'pending'`,
		workspaceID, email); err != nil {
		return nil, fmt.Errorf("supersede pending invite: %w", err)
	}

	inv, err := scanInvitation(s.Pool.QueryRow(ctx,
		`INSERT INTO invitations (workspace_id, email, role, token, invited_by, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING `+invitationColumns,
		workspaceID, email, role, token, invitedBy, expiresAt))
	if err != nil {
		return nil, fmt.Errorf("create invitation: %w", err)
	}
	return inv, nil
}

// ListInvitations returns invitations for a workspace, optionally only pending
// ones, with the inviter's email for display.
func (s *AuthStore) ListInvitations(ctx context.Context, workspaceID int64, pendingOnly bool) ([]Invitation, error) {
	where := "i.workspace_id = $1"
	if pendingOnly {
		where += " AND i.status = 'pending'"
	}
	rows, err := s.Pool.Query(ctx,
		`SELECT `+prefixColumns("i", invitationColumns)+`, COALESCE(u.email, '')
		 FROM invitations i
		 LEFT JOIN users u ON u.id = i.invited_by
		 WHERE `+where+`
		 ORDER BY i.created_at DESC`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list invitations: %w", err)
	}
	defer rows.Close()

	var invites []Invitation
	for rows.Next() {
		var inv Invitation
		if err := rows.Scan(&inv.ID, &inv.WorkspaceID, &inv.Email, &inv.Role, &inv.Token,
			&inv.InvitedBy, &inv.Status, &inv.ExpiresAt, &inv.CreatedAt, &inv.AcceptedAt,
			&inv.InvitedByEmail); err != nil {
			return nil, err
		}
		invites = append(invites, inv)
	}
	return invites, rows.Err()
}

// GetInvitationByToken returns a pending, non-expired invitation with the
// workspace name for display. Returns an error if missing, revoked, accepted, or
// expired.
func (s *AuthStore) GetInvitationByToken(ctx context.Context, token string) (*Invitation, error) {
	inv := &Invitation{}
	err := s.Pool.QueryRow(ctx,
		`SELECT `+prefixColumns("i", invitationColumns)+`, w.name
		 FROM invitations i JOIN workspaces w ON w.id = i.workspace_id
		 WHERE i.token = $1`, token,
	).Scan(&inv.ID, &inv.WorkspaceID, &inv.Email, &inv.Role, &inv.Token, &inv.InvitedBy,
		&inv.Status, &inv.ExpiresAt, &inv.CreatedAt, &inv.AcceptedAt, &inv.WorkspaceName)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("invitation not found")
		}
		return nil, fmt.Errorf("get invitation: %w", err)
	}
	switch inv.Status {
	case InviteStatusAccepted:
		return nil, errors.New("invitation already accepted")
	case InviteStatusRevoked:
		return nil, errors.New("invitation has been revoked")
	case InviteStatusExpired:
		return nil, errors.New("invitation has expired")
	}
	if time.Now().After(inv.ExpiresAt) {
		return nil, errors.New("invitation has expired")
	}
	return inv, nil
}

// AcceptInvitation validates the token, creates (or upgrades) the user's
// membership, and marks the invite accepted — all in one transaction (Phase 15.7).
// The caller must be the authenticated user whose email matches the invite
// (Phase 15.8: the account may have just been created during signup).
func (s *AuthStore) AcceptInvitation(ctx context.Context, token string, userID int64, userEmail string) (*Invitation, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("accept invitation: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	inv := &Invitation{}
	err = tx.QueryRow(ctx,
		`SELECT `+invitationColumns+` FROM invitations WHERE token = $1 FOR UPDATE`, token,
	).Scan(&inv.ID, &inv.WorkspaceID, &inv.Email, &inv.Role, &inv.Token, &inv.InvitedBy,
		&inv.Status, &inv.ExpiresAt, &inv.CreatedAt, &inv.AcceptedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("invitation not found")
		}
		return nil, fmt.Errorf("accept invitation: %w", err)
	}
	if inv.Status != InviteStatusPending {
		return nil, fmt.Errorf("invitation is %s", inv.Status)
	}
	if time.Now().After(inv.ExpiresAt) {
		return nil, errors.New("invitation has expired")
	}
	// The signed-in account must match the invited email.
	if !strings.EqualFold(strings.TrimSpace(userEmail), inv.Email) {
		return nil, errors.New("this invitation was sent to a different email address")
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO memberships (user_id, workspace_id, role)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (user_id, workspace_id) DO UPDATE SET role = EXCLUDED.role`,
		userID, inv.WorkspaceID, inv.Role); err != nil {
		return nil, fmt.Errorf("accept invitation (membership): %w", err)
	}
	if _, err := tx.Exec(ctx,
		`UPDATE invitations SET status = 'accepted', accepted_at = NOW() WHERE id = $1`,
		inv.ID); err != nil {
		return nil, fmt.Errorf("accept invitation (mark): %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("accept invitation: commit: %w", err)
	}
	inv.Status = InviteStatusAccepted
	return inv, nil
}

// ResendInvitation issues a fresh token and expiry for a pending invite so a new
// email can be sent (Phase 15.9). Returns the refreshed invitation.
func (s *AuthStore) ResendInvitation(ctx context.Context, workspaceID, inviteID int64) (*Invitation, error) {
	token, err := GenerateSessionID()
	if err != nil {
		return nil, err
	}
	inv, err := scanInvitation(s.Pool.QueryRow(ctx,
		`UPDATE invitations
		 SET token = $1, expires_at = $2, status = 'pending', created_at = NOW()
		 WHERE id = $3 AND workspace_id = $4 AND status IN ('pending', 'expired')
		 RETURNING `+invitationColumns,
		token, time.Now().Add(InviteTTL), inviteID, workspaceID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("invitation not found or not resendable")
		}
		return nil, fmt.Errorf("resend invitation: %w", err)
	}
	return inv, nil
}

// RevokeInvitation cancels a pending invitation (Phase 15.9).
func (s *AuthStore) RevokeInvitation(ctx context.Context, workspaceID, inviteID int64) error {
	result, err := s.Pool.Exec(ctx,
		`UPDATE invitations SET status = 'revoked'
		 WHERE id = $1 AND workspace_id = $2 AND status = 'pending'`,
		inviteID, workspaceID)
	if err != nil {
		return fmt.Errorf("revoke invitation: %w", err)
	}
	if result.RowsAffected() == 0 {
		return errors.New("invitation not found or not pending")
	}
	return nil
}

// SweepExpiredInvitations marks past-due pending invitations as expired and
// returns the number swept (Phase 15.9). Intended to run periodically.
func (s *AuthStore) SweepExpiredInvitations(ctx context.Context) (int64, error) {
	result, err := s.Pool.Exec(ctx,
		`UPDATE invitations SET status = 'expired'
		 WHERE status = 'pending' AND expires_at < NOW()`)
	if err != nil {
		return 0, fmt.Errorf("sweep expired invitations: %w", err)
	}
	return result.RowsAffected(), nil
}

// prefixColumns rewrites a comma-separated column list to be table-qualified,
// e.g. prefixColumns("i", "id, email") -> "i.id, i.email".
func prefixColumns(alias, cols string) string {
	parts := strings.Split(cols, ",")
	for idx, p := range parts {
		parts[idx] = alias + "." + strings.TrimSpace(p)
	}
	return strings.Join(parts, ", ")
}
