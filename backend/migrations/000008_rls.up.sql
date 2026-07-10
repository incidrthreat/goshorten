-- Phase 14.7: Row-Level Security as a defense-in-depth backstop to the
-- application-level workspace scoping done in the repository layer.
--
-- Scope: RLS is enabled on `urls` only. `urls` is the canonical tenant object and
-- every query against it flows through the PostgresStore repository, which wraps
-- workspace-scoped operations in a transaction that sets `app.current_workspace_id`
-- and global operations (redirects, previews, admin operator views) in a
-- transaction that sets `app.bypass_rls = 'on'`. clicks/tags/api_keys/domains stay
-- application-scoped because they are also read in pre-workspace/global contexts
-- (async click logging, API-key validation, redirect domain resolution) where a
-- hard RLS gate would break those paths. See backend/data/postgres.go.
--
-- The policy is default-deny: with no GUC set, current_setting(..., true) returns
-- NULL and the row comparison yields NULL (not true), so unscoped connections see
-- nothing. This only bites a non-superuser role; a superuser/BYPASSRLS login is
-- unaffected, so run the app as the non-privileged GOSHORTEN_POSTGRES_USER.

ALTER TABLE urls ENABLE ROW LEVEL SECURITY;
ALTER TABLE urls FORCE ROW LEVEL SECURITY;

CREATE POLICY urls_workspace_isolation ON urls
    USING (
        current_setting('app.bypass_rls', true) = 'on'
        OR workspace_id = NULLIF(current_setting('app.current_workspace_id', true), '')::bigint
    )
    WITH CHECK (
        current_setting('app.bypass_rls', true) = 'on'
        OR workspace_id = NULLIF(current_setting('app.current_workspace_id', true), '')::bigint
    );
