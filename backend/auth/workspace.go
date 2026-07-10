package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// Workspace is the billable tenant unit (Phase 14.1).
type Workspace struct {
	ID        int64
	Name      string
	Slug      string
	OwnerID   int64
	Plan      string
	CreatedAt time.Time
}

const workspaceColumns = `id, name, slug, owner_id, plan, created_at`

func scanWorkspace(row interface{ Scan(...any) error }) (*Workspace, error) {
	w := &Workspace{}
	if err := row.Scan(&w.ID, &w.Name, &w.Slug, &w.OwnerID, &w.Plan, &w.CreatedAt); err != nil {
		return nil, err
	}
	return w, nil
}

// CreateWorkspace creates a new workspace owned by ownerID, deriving a unique slug
// from the name.
func (s *AuthStore) CreateWorkspace(ctx context.Context, ownerID int64, name string) (*Workspace, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("workspace name is required")
	}

	base := slugify(name)
	if base == "" {
		base = "workspace"
	}

	for attempt := 0; attempt < 8; attempt++ {
		slug := base
		if attempt > 0 {
			slug = fmt.Sprintf("%s-%d", base, attempt+1)
		}
		w := &Workspace{}
		err := s.Pool.QueryRow(ctx,
			`INSERT INTO workspaces (name, slug, owner_id, plan) VALUES ($1, $2, $3, 'free')
			 RETURNING `+workspaceColumns,
			name, slug, ownerID,
		).Scan(&w.ID, &w.Name, &w.Slug, &w.OwnerID, &w.Plan, &w.CreatedAt)
		if err == nil {
			return w, nil
		}
		if isUniqueViolationErr(err) {
			continue // slug collision — try the next suffix
		}
		return nil, fmt.Errorf("create workspace: %w", err)
	}
	return nil, errors.New("could not allocate a unique workspace slug")
}

// GetWorkspace returns a workspace by id.
func (s *AuthStore) GetWorkspace(ctx context.Context, id int64) (*Workspace, error) {
	w, err := scanWorkspace(s.Pool.QueryRow(ctx,
		`SELECT `+workspaceColumns+` FROM workspaces WHERE id = $1`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("workspace not found")
		}
		return nil, fmt.Errorf("get workspace: %w", err)
	}
	return w, nil
}

// ListWorkspacesForUser returns the workspaces a user can access via membership
// (Phase 15), plus their owned workspaces and default workspace as safety nets
// for any account that predates the memberships backfill.
func (s *AuthStore) ListWorkspacesForUser(ctx context.Context, userID int64) ([]Workspace, error) {
	rows, err := s.Pool.Query(ctx,
		`SELECT DISTINCT `+prefixColumns("w", workspaceColumns)+`
		 FROM workspaces w
		 WHERE w.owner_id = $1
		    OR w.id = (SELECT default_workspace_id FROM users WHERE id = $1)
		    OR EXISTS (SELECT 1 FROM memberships m
		               WHERE m.workspace_id = w.id AND m.user_id = $1)
		 ORDER BY w.created_at`, userID)
	if err != nil {
		return nil, fmt.Errorf("list workspaces: %w", err)
	}
	defer rows.Close()

	var workspaces []Workspace
	for rows.Next() {
		w := Workspace{}
		if err := rows.Scan(&w.ID, &w.Name, &w.Slug, &w.OwnerID, &w.Plan, &w.CreatedAt); err != nil {
			return nil, err
		}
		workspaces = append(workspaces, w)
	}
	return workspaces, rows.Err()
}

// UserCanAccessWorkspace reports whether a user may operate within a workspace.
// Access is granted by an active membership (Phase 15), with ownership and the
// user's default workspace accepted as transitional bridges for legacy accounts.
func (s *AuthStore) UserCanAccessWorkspace(ctx context.Context, userID, workspaceID int64) (bool, error) {
	if workspaceID <= 0 {
		return false, nil
	}
	var ok bool
	err := s.Pool.QueryRow(ctx,
		`SELECT EXISTS (
			SELECT 1 FROM memberships m
			  WHERE m.workspace_id = $2 AND m.user_id = $1 AND m.status = 'active'
			UNION
			SELECT 1 FROM workspaces w WHERE w.id = $2 AND w.owner_id = $1
			UNION
			SELECT 1 FROM users u WHERE u.id = $1 AND u.default_workspace_id = $2
		 )`, userID, workspaceID).Scan(&ok)
	if err != nil {
		return false, fmt.Errorf("check workspace access: %w", err)
	}
	return ok, nil
}

// WorkspaceRole returns the caller's effective role within a workspace. It prefers
// the membership role; if there is no membership row but the user owns the
// workspace or it is their default (legacy bridge), it reports "owner"/"member"
// respectively. Returns "" when the user has no access.
func (s *AuthStore) WorkspaceRole(ctx context.Context, userID, workspaceID int64) (string, error) {
	if workspaceID <= 0 {
		return "", nil
	}
	m, err := s.GetMembership(ctx, userID, workspaceID)
	if err != nil {
		return "", err
	}
	if m != nil {
		return m.Role, nil
	}
	// No membership row — fall back to legacy signals.
	var role string
	err = s.Pool.QueryRow(ctx,
		`SELECT CASE
		          WHEN w.owner_id = $1 THEN 'owner'
		          WHEN u.default_workspace_id = $2 THEN 'member'
		          ELSE ''
		        END
		 FROM workspaces w, users u
		 WHERE w.id = $2 AND u.id = $1`, userID, workspaceID).Scan(&role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("workspace role: %w", err)
	}
	return role, nil
}

// DeleteWorkspace removes a workspace owned by ownerID. The owner's personal
// (default) workspace cannot be deleted.
func (s *AuthStore) DeleteWorkspace(ctx context.Context, ownerID, workspaceID int64) error {
	var defaultWS *int64
	if err := s.Pool.QueryRow(ctx,
		`SELECT default_workspace_id FROM users WHERE id = $1`, ownerID).Scan(&defaultWS); err != nil {
		return fmt.Errorf("delete workspace: %w", err)
	}
	if defaultWS != nil && *defaultWS == workspaceID {
		return errors.New("cannot delete your personal workspace")
	}

	result, err := s.Pool.Exec(ctx,
		`DELETE FROM workspaces WHERE id = $1 AND owner_id = $2`, workspaceID, ownerID)
	if err != nil {
		return fmt.Errorf("delete workspace: %w", err)
	}
	if result.RowsAffected() == 0 {
		return errors.New("workspace not found or not owned by you")
	}
	return nil
}

// SetActiveWorkspace persists the active workspace for a session (Phase 14.8).
func (s *AuthStore) SetActiveWorkspace(ctx context.Context, sessionID string, workspaceID int64) error {
	if sessionID == "" {
		return nil
	}
	_, err := s.Pool.Exec(ctx,
		`UPDATE sessions SET active_workspace_id = $1 WHERE id = $2`, workspaceID, sessionID)
	if err != nil {
		return fmt.Errorf("set active workspace: %w", err)
	}
	return nil
}

// ResolveActiveWorkspace determines the active workspace for a request, in order:
//  1. an explicit, accessible header override (X-Workspace-Id),
//  2. the session's last-switched workspace (if still accessible),
//  3. the user's default workspace.
// Returns 0 if none can be resolved.
func (s *AuthStore) ResolveActiveWorkspace(ctx context.Context, userID int64, sessionID string, headerWorkspaceID int64) (int64, error) {
	if headerWorkspaceID > 0 {
		ok, err := s.UserCanAccessWorkspace(ctx, userID, headerWorkspaceID)
		if err != nil {
			return 0, err
		}
		if ok {
			return headerWorkspaceID, nil
		}
	}

	if sessionID != "" {
		var sessionWS *int64
		err := s.Pool.QueryRow(ctx,
			`SELECT active_workspace_id FROM sessions WHERE id = $1`, sessionID).Scan(&sessionWS)
		if err == nil && sessionWS != nil {
			ok, err := s.UserCanAccessWorkspace(ctx, userID, *sessionWS)
			if err != nil {
				return 0, err
			}
			if ok {
				return *sessionWS, nil
			}
		}
	}

	var defaultWS *int64
	if err := s.Pool.QueryRow(ctx,
		`SELECT default_workspace_id FROM users WHERE id = $1`, userID).Scan(&defaultWS); err != nil {
		return 0, fmt.Errorf("resolve workspace: %w", err)
	}
	if defaultWS != nil {
		return *defaultWS, nil
	}
	return 0, nil
}

// EnsureDefaultWorkspace returns the id of the instance's default workspace used
// for anonymous/legacy writes. It prefers the backfilled "default" slug, then the
// earliest workspace. Returns 0 if no workspace exists yet.
func (s *AuthStore) EnsureDefaultWorkspace(ctx context.Context) (int64, error) {
	var id int64
	err := s.Pool.QueryRow(ctx,
		`SELECT id FROM workspaces WHERE slug = 'default' ORDER BY id LIMIT 1`).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("ensure default workspace: %w", err)
	}
	err = s.Pool.QueryRow(ctx,
		`SELECT id FROM workspaces ORDER BY id LIMIT 1`).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil
		}
		return 0, fmt.Errorf("ensure default workspace: %w", err)
	}
	return id, nil
}

// slugify converts a name into a URL-safe slug.
func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	lastDash := false
	for _, r := range s {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash && b.Len() > 0 {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

// isUniqueViolationErr reports a Postgres unique-constraint violation (SQLSTATE 23505).
func isUniqueViolationErr(err error) bool {
	var pgErr interface{ SQLState() string }
	if errors.As(err, &pgErr) {
		return pgErr.SQLState() == "23505"
	}
	return false
}
