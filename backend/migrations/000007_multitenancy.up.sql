-- Phase 14: Multi-Tenancy Foundation (Workspaces & Tenant Isolation)

-- 14.1 workspaces table — the billable tenant unit
CREATE TABLE IF NOT EXISTS workspaces (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    slug        VARCHAR(100) UNIQUE NOT NULL,
    owner_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plan        VARCHAR(50) NOT NULL DEFAULT 'free',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_workspaces_owner ON workspaces(owner_id);

-- Each user has a "default" workspace (their personal one) used as the active
-- workspace fallback when no session/header override is present.
ALTER TABLE users ADD COLUMN IF NOT EXISTS default_workspace_id BIGINT
    REFERENCES workspaces(id) ON DELETE SET NULL;

-- Sessions remember which workspace the user last switched to (14.8).
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS active_workspace_id BIGINT
    REFERENCES workspaces(id) ON DELETE SET NULL;

-- 14.2 workspace_id foreign keys on tenant-scoped tables.
-- Kept nullable so existing/anonymous rows and pre-bootstrap inserts never fail;
-- the application always sets workspace_id on new rows.
ALTER TABLE urls     ADD COLUMN IF NOT EXISTS workspace_id BIGINT REFERENCES workspaces(id) ON DELETE CASCADE;
ALTER TABLE clicks   ADD COLUMN IF NOT EXISTS workspace_id BIGINT REFERENCES workspaces(id) ON DELETE CASCADE;
ALTER TABLE tags     ADD COLUMN IF NOT EXISTS workspace_id BIGINT REFERENCES workspaces(id) ON DELETE CASCADE;
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS workspace_id BIGINT REFERENCES workspaces(id) ON DELETE CASCADE;
ALTER TABLE domains  ADD COLUMN IF NOT EXISTS workspace_id BIGINT REFERENCES workspaces(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_urls_workspace     ON urls(workspace_id);
CREATE INDEX IF NOT EXISTS idx_clicks_workspace   ON clicks(workspace_id);
CREATE INDEX IF NOT EXISTS idx_tags_workspace     ON tags(workspace_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_workspace ON api_keys(workspace_id);
CREATE INDEX IF NOT EXISTS idx_domains_workspace  ON domains(workspace_id);

-- Tags are now unique per workspace rather than globally.
ALTER TABLE tags DROP CONSTRAINT IF EXISTS tags_name_key;
CREATE UNIQUE INDEX IF NOT EXISTS idx_tags_workspace_name ON tags(workspace_id, name);

-- 14.3 Backfill: move all existing rows into a single default workspace owned by
-- the earliest admin (falling back to the earliest user). Every existing user is
-- pointed at that workspace so no account is ever workspace-less.
DO $$
DECLARE
    default_owner BIGINT;
    default_ws    BIGINT;
BEGIN
    SELECT id INTO default_owner FROM users WHERE role = 'admin' ORDER BY id LIMIT 1;
    IF default_owner IS NULL THEN
        SELECT id INTO default_owner FROM users ORDER BY id LIMIT 1;
    END IF;

    -- Nothing to backfill on a fresh install (no users yet).
    IF default_owner IS NULL THEN
        RETURN;
    END IF;

    INSERT INTO workspaces (name, slug, owner_id, plan)
    VALUES ('Default', 'default', default_owner, 'free')
    RETURNING id INTO default_ws;

    UPDATE urls     SET workspace_id = default_ws WHERE workspace_id IS NULL;
    UPDATE clicks   SET workspace_id = default_ws WHERE workspace_id IS NULL;
    UPDATE tags     SET workspace_id = default_ws WHERE workspace_id IS NULL;
    UPDATE api_keys SET workspace_id = default_ws WHERE workspace_id IS NULL;
    UPDATE domains  SET workspace_id = default_ws WHERE workspace_id IS NULL;

    -- Every existing user shares the default workspace as their landing place,
    -- preserving the pre-multitenancy "everyone could see the shared data" behavior.
    UPDATE users SET default_workspace_id = default_ws WHERE default_workspace_id IS NULL;
END $$;
