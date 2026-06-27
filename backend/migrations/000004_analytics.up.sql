-- Phase 5: Analytics & Visit Tracking

-- Add geo and parsed UA columns to clicks table
ALTER TABLE clicks
    ADD COLUMN IF NOT EXISTS country     VARCHAR(2),
    ADD COLUMN IF NOT EXISTS city        VARCHAR(255),
    ADD COLUMN IF NOT EXISTS device_type VARCHAR(20),
    ADD COLUMN IF NOT EXISTS browser     VARCHAR(100),
    ADD COLUMN IF NOT EXISTS os          VARCHAR(100),
    ADD COLUMN IF NOT EXISTS is_bot      BOOLEAN NOT NULL DEFAULT FALSE;

-- Indexes for analytics aggregation
CREATE INDEX IF NOT EXISTS idx_clicks_country    ON clicks(country)    WHERE country IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_clicks_browser    ON clicks(browser)    WHERE browser IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_clicks_os         ON clicks(os)         WHERE os IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_clicks_is_bot     ON clicks(is_bot);
CREATE INDEX IF NOT EXISTS idx_clicks_referer    ON clicks(referer)    WHERE referer IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_clicks_ip_address ON clicks(ip_address) WHERE ip_address IS NOT NULL;

-- Composite index for time-series aggregation per URL
CREATE INDEX IF NOT EXISTS idx_clicks_url_date ON clicks(url_id, clicked_at);

-- Orphan visits: clicks to codes that don't exist or are expired/inactive
CREATE TABLE IF NOT EXISTS orphan_visits (
    id          BIGSERIAL PRIMARY KEY,
    code        VARCHAR(100) NOT NULL,
    visited_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ip_address  INET,
    user_agent  TEXT,
    referer     TEXT,
    country     VARCHAR(2),
    city        VARCHAR(255),
    is_bot      BOOLEAN NOT NULL DEFAULT FALSE
);
CREATE INDEX IF NOT EXISTS idx_orphan_visits_code ON orphan_visits(code);
CREATE INDEX IF NOT EXISTS idx_orphan_visits_visited_at ON orphan_visits(visited_at);
