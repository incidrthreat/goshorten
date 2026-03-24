CREATE TABLE IF NOT EXISTS system_settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL DEFAULT ''
);

INSERT INTO system_settings (key, value) VALUES ('password_login_enabled', 'true')
ON CONFLICT (key) DO NOTHING;
