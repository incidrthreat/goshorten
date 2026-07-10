-- Reverse Phase 14 multi-tenancy schema changes.

DROP INDEX IF EXISTS idx_tags_workspace_name;
-- Restore the global-unique tag name constraint.
ALTER TABLE tags ADD CONSTRAINT tags_name_key UNIQUE (name);

DROP INDEX IF EXISTS idx_urls_workspace;
DROP INDEX IF EXISTS idx_clicks_workspace;
DROP INDEX IF EXISTS idx_tags_workspace;
DROP INDEX IF EXISTS idx_api_keys_workspace;
DROP INDEX IF EXISTS idx_domains_workspace;

ALTER TABLE sessions DROP COLUMN IF EXISTS active_workspace_id;
ALTER TABLE users    DROP COLUMN IF EXISTS default_workspace_id;

ALTER TABLE urls     DROP COLUMN IF EXISTS workspace_id;
ALTER TABLE clicks   DROP COLUMN IF EXISTS workspace_id;
ALTER TABLE tags     DROP COLUMN IF EXISTS workspace_id;
ALTER TABLE api_keys DROP COLUMN IF EXISTS workspace_id;
ALTER TABLE domains  DROP COLUMN IF EXISTS workspace_id;

DROP INDEX IF EXISTS idx_workspaces_owner;
DROP TABLE IF EXISTS workspaces;
