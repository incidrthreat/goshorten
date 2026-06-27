-- Extend users table for auth
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS role VARCHAR(20) NOT NULL DEFAULT 'user';
ALTER TABLE users ADD COLUMN IF NOT EXISTS oidc_subject VARCHAR(255);
ALTER TABLE users ADD COLUMN IF NOT EXISTS oidc_issuer VARCHAR(512);
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT TRUE;

-- Unique constraint: one OIDC identity per user per provider
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_oidc
    ON users(oidc_issuer, oidc_subject) WHERE oidc_issuer IS NOT NULL;

-- OIDC provider configuration (multi-provider ready)
CREATE TABLE IF NOT EXISTS oidc_providers (
    id            BIGSERIAL PRIMARY KEY,
    name          VARCHAR(100) UNIQUE NOT NULL,
    issuer_url    VARCHAR(512) NOT NULL,
    client_id     VARCHAR(255) NOT NULL,
    client_secret VARCHAR(512) NOT NULL,
    redirect_uri  VARCHAR(512) NOT NULL,
    scopes        TEXT NOT NULL DEFAULT 'openid email profile',
    is_enabled    BOOLEAN NOT NULL DEFAULT TRUE,
    auto_register BOOLEAN NOT NULL DEFAULT TRUE,
    default_role  VARCHAR(20) NOT NULL DEFAULT 'user',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Extend api_keys table with scopes
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS scopes TEXT NOT NULL DEFAULT 'urls:read,urls:write';

-- Sessions table for OIDC login sessions
CREATE TABLE IF NOT EXISTS sessions (
    id          VARCHAR(64) PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at  TIMESTAMPTZ NOT NULL,
    ip_address  INET
);
CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);
