-- Phase 15: Memberships, Roles & Invitations

-- 15.1 memberships table — moves role from a global user property to a
-- per-workspace association. The global users.role remains the platform/operator
-- flag (service owner); membership.role governs what a user can do *inside* a
-- workspace (Phase 14.9 separation).
--
-- 15.3 workspace roles: owner (full control incl. billing/ownership), admin
-- (manage members + content), member (create/edit content), viewer (read-only).
CREATE TABLE IF NOT EXISTS memberships (
    id           BIGSERIAL PRIMARY KEY,
    user_id      BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    workspace_id BIGINT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    role         VARCHAR(20) NOT NULL DEFAULT 'member'
                 CHECK (role IN ('owner', 'admin', 'member', 'viewer')),
    status       VARCHAR(20) NOT NULL DEFAULT 'active'
                 CHECK (status IN ('active', 'suspended')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, workspace_id)
);
CREATE INDEX IF NOT EXISTS idx_memberships_workspace ON memberships(workspace_id);
CREATE INDEX IF NOT EXISTS idx_memberships_user ON memberships(user_id);

-- 15.6 invitations table — a pending invite grants membership at the given role
-- once accepted. The token is a random secret embedded in the accept link.
CREATE TABLE IF NOT EXISTS invitations (
    id           BIGSERIAL PRIMARY KEY,
    workspace_id BIGINT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    email        VARCHAR(255) NOT NULL,
    role         VARCHAR(20) NOT NULL DEFAULT 'member'
                 CHECK (role IN ('admin', 'member', 'viewer')),
    token        VARCHAR(64) NOT NULL UNIQUE,
    invited_by   BIGINT REFERENCES users(id) ON DELETE SET NULL,
    status       VARCHAR(20) NOT NULL DEFAULT 'pending'
                 CHECK (status IN ('pending', 'accepted', 'revoked', 'expired')),
    expires_at   TIMESTAMPTZ NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    accepted_at  TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_invitations_workspace ON invitations(workspace_id);
CREATE INDEX IF NOT EXISTS idx_invitations_email ON invitations(lower(email));
-- At most one *pending* invite per (workspace, email); resends replace in place.
CREATE UNIQUE INDEX IF NOT EXISTS idx_invitations_pending
    ON invitations(workspace_id, lower(email)) WHERE status = 'pending';

-- 15.4 backfill: every existing workspace's owner becomes an 'owner' member so the
-- "at least one Owner" invariant holds from the start.
INSERT INTO memberships (user_id, workspace_id, role)
SELECT owner_id, id, 'owner' FROM workspaces
ON CONFLICT (user_id, workspace_id) DO NOTHING;

-- Legacy accounts share the backfilled "Default" workspace via default_workspace_id
-- (Phase 14.3). Give each such non-owner user a 'member' membership so they retain
-- the access they had before memberships existed.
INSERT INTO memberships (user_id, workspace_id, role)
SELECT u.id, u.default_workspace_id, 'member'
FROM users u
WHERE u.default_workspace_id IS NOT NULL
ON CONFLICT (user_id, workspace_id) DO NOTHING;
