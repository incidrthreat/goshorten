-- Phase 12: Settings, Account, and Security

-- 12.4 Persistent theme preference per user
ALTER TABLE users ADD COLUMN IF NOT EXISTS theme VARCHAR(50) NOT NULL DEFAULT 'system';

-- 12.2 Enhance sessions table for full session management
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS user_agent TEXT;
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS label     VARCHAR(255);
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS revoked   BOOLEAN NOT NULL DEFAULT FALSE;
CREATE INDEX IF NOT EXISTS idx_sessions_active ON sessions(user_id, expires_at) WHERE NOT revoked;

-- 12.2 Sign-in history
CREATE TABLE IF NOT EXISTS sign_in_events (
    id           BIGSERIAL PRIMARY KEY,
    user_id      BIGINT REFERENCES users(id) ON DELETE SET NULL,
    ip_address   INET,
    user_agent   TEXT,
    success      BOOLEAN NOT NULL DEFAULT TRUE,
    signed_in_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_sign_in_events_user   ON sign_in_events(user_id);
CREATE INDEX IF NOT EXISTS idx_sign_in_events_time   ON sign_in_events(signed_in_at DESC);
