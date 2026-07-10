package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// Workspace role constants (Phase 15.3). Ordered by privilege.
const (
	RoleOwner  = "owner"  // full control incl. ownership transfer / workspace deletion
	RoleAdmin  = "admin"  // manage members, invites, and all content
	RoleMember = "member" // create/edit content
	RoleViewer = "viewer" // read-only
)

// roleRank orders workspace roles by privilege for comparison.
var roleRank = map[string]int{
	RoleViewer: 1,
	RoleMember: 2,
	RoleAdmin:  3,
	RoleOwner:  4,
}

// ValidWorkspaceRole reports whether role is a recognized workspace role.
func ValidWorkspaceRole(role string) bool {
	_, ok := roleRank[role]
	return ok
}

// RoleAtLeast reports whether have meets or exceeds the want privilege level.
func RoleAtLeast(have, want string) bool {
	return roleRank[have] >= roleRank[want] && roleRank[have] > 0
}

// ErrLastOwner is returned when an operation would leave a workspace with no owner.
var ErrLastOwner = errors.New("workspace must have at least one owner")

// Membership associates a user with a workspace at a given role (Phase 15.1).
type Membership struct {
	ID          int64
	UserID      int64
	WorkspaceID int64
	Role        string
	Status      string
	CreatedAt   time.Time
	// Joined user fields (populated by ListMembers).
	Email string
	Name  string
}

// GetMembership returns the caller's membership in a workspace, or nil if none.
func (s *AuthStore) GetMembership(ctx context.Context, userID, workspaceID int64) (*Membership, error) {
	m := &Membership{}
	err := s.Pool.QueryRow(ctx,
		`SELECT id, user_id, workspace_id, role, status, created_at
		 FROM memberships WHERE user_id = $1 AND workspace_id = $2`,
		userID, workspaceID,
	).Scan(&m.ID, &m.UserID, &m.WorkspaceID, &m.Role, &m.Status, &m.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get membership: %w", err)
	}
	return m, nil
}

// ListMembers returns all memberships in a workspace with the member's identity.
func (s *AuthStore) ListMembers(ctx context.Context, workspaceID int64) ([]Membership, error) {
	rows, err := s.Pool.Query(ctx,
		`SELECT m.id, m.user_id, m.workspace_id, m.role, m.status, m.created_at,
		        u.email, COALESCE(u.name, '')
		 FROM memberships m
		 JOIN users u ON u.id = m.user_id
		 WHERE m.workspace_id = $1
		 ORDER BY CASE m.role
		            WHEN 'owner' THEN 0 WHEN 'admin' THEN 1
		            WHEN 'member' THEN 2 ELSE 3 END, u.email`,
		workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	defer rows.Close()

	var members []Membership
	for rows.Next() {
		var m Membership
		if err := rows.Scan(&m.ID, &m.UserID, &m.WorkspaceID, &m.Role, &m.Status,
			&m.CreatedAt, &m.Email, &m.Name); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

// UpsertMembership creates or updates a user's membership in a workspace. Used by
// signup (owner of personal workspace) and invite acceptance.
func (s *AuthStore) UpsertMembership(ctx context.Context, userID, workspaceID int64, role string) error {
	if !ValidWorkspaceRole(role) {
		return fmt.Errorf("invalid role %q", role)
	}
	_, err := s.Pool.Exec(ctx,
		`INSERT INTO memberships (user_id, workspace_id, role)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (user_id, workspace_id) DO UPDATE SET role = EXCLUDED.role`,
		userID, workspaceID, role)
	if err != nil {
		return fmt.Errorf("upsert membership: %w", err)
	}
	return nil
}

// countOwners returns the number of owner memberships in a workspace, using the
// supplied querier so it can run inside a transaction.
func countOwners(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, workspaceID int64) (int, error) {
	var n int
	err := q.QueryRow(ctx,
		`SELECT COUNT(*) FROM memberships WHERE workspace_id = $1 AND role = 'owner'`,
		workspaceID).Scan(&n)
	return n, err
}

// UpdateMemberRole changes a member's role, enforcing the "at least one owner"
// invariant (Phase 15.4): the last remaining owner cannot be demoted.
func (s *AuthStore) UpdateMemberRole(ctx context.Context, workspaceID, userID int64, role string) error {
	if !ValidWorkspaceRole(role) {
		return fmt.Errorf("invalid role %q", role)
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("update member role: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var current string
	err = tx.QueryRow(ctx,
		`SELECT role FROM memberships WHERE workspace_id = $1 AND user_id = $2 FOR UPDATE`,
		workspaceID, userID).Scan(&current)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errors.New("member not found")
		}
		return fmt.Errorf("update member role: %w", err)
	}
	if current == role {
		return nil
	}

	// Demoting an owner: ensure another owner remains.
	if current == RoleOwner && role != RoleOwner {
		owners, err := countOwners(ctx, tx, workspaceID)
		if err != nil {
			return fmt.Errorf("count owners: %w", err)
		}
		if owners <= 1 {
			return ErrLastOwner
		}
	}

	if _, err := tx.Exec(ctx,
		`UPDATE memberships SET role = $1 WHERE workspace_id = $2 AND user_id = $3`,
		role, workspaceID, userID); err != nil {
		return fmt.Errorf("update member role: %w", err)
	}
	// Keep workspaces.owner_id pointing at an actual owner when promoting.
	if role == RoleOwner {
		if _, err := tx.Exec(ctx,
			`UPDATE workspaces SET owner_id = $1 WHERE id = $2 AND owner_id NOT IN
			   (SELECT user_id FROM memberships WHERE workspace_id = $2 AND role = 'owner')`,
			userID, workspaceID); err != nil {
			return fmt.Errorf("sync workspace owner: %w", err)
		}
	}
	return tx.Commit(ctx)
}

// RemoveMember deletes a user's membership, enforcing the "at least one owner"
// invariant: the last owner cannot be removed (transfer ownership first).
func (s *AuthStore) RemoveMember(ctx context.Context, workspaceID, userID int64) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("remove member: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var current string
	err = tx.QueryRow(ctx,
		`SELECT role FROM memberships WHERE workspace_id = $1 AND user_id = $2 FOR UPDATE`,
		workspaceID, userID).Scan(&current)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errors.New("member not found")
		}
		return fmt.Errorf("remove member: %w", err)
	}

	if current == RoleOwner {
		owners, err := countOwners(ctx, tx, workspaceID)
		if err != nil {
			return fmt.Errorf("count owners: %w", err)
		}
		if owners <= 1 {
			return ErrLastOwner
		}
	}

	if _, err := tx.Exec(ctx,
		`DELETE FROM memberships WHERE workspace_id = $1 AND user_id = $2`,
		workspaceID, userID); err != nil {
		return fmt.Errorf("remove member: %w", err)
	}
	return tx.Commit(ctx)
}

// TransferOwnership makes newOwnerID an owner of the workspace and updates
// workspaces.owner_id (Phase 15.5). The previous owner is demoted to admin unless
// keepPrevAsOwner is set. newOwnerID must already be a member.
func (s *AuthStore) TransferOwnership(ctx context.Context, workspaceID, currentOwnerID, newOwnerID int64) error {
	if newOwnerID == currentOwnerID {
		return errors.New("new owner is already the owner")
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("transfer ownership: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var targetRole string
	err = tx.QueryRow(ctx,
		`SELECT role FROM memberships WHERE workspace_id = $1 AND user_id = $2 FOR UPDATE`,
		workspaceID, newOwnerID).Scan(&targetRole)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errors.New("new owner must be a member of the workspace")
		}
		return fmt.Errorf("transfer ownership: %w", err)
	}

	if _, err := tx.Exec(ctx,
		`UPDATE memberships SET role = 'owner' WHERE workspace_id = $1 AND user_id = $2`,
		workspaceID, newOwnerID); err != nil {
		return fmt.Errorf("transfer ownership (promote): %w", err)
	}
	// Demote the previous owner to admin (they keep management access).
	if _, err := tx.Exec(ctx,
		`UPDATE memberships SET role = 'admin'
		 WHERE workspace_id = $1 AND user_id = $2 AND role = 'owner'`,
		workspaceID, currentOwnerID); err != nil {
		return fmt.Errorf("transfer ownership (demote): %w", err)
	}
	if _, err := tx.Exec(ctx,
		`UPDATE workspaces SET owner_id = $1 WHERE id = $2`,
		newOwnerID, workspaceID); err != nil {
		return fmt.Errorf("transfer ownership (owner_id): %w", err)
	}
	return tx.Commit(ctx)
}
