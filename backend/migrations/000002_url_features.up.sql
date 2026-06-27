-- Custom slug support: allow user-provided codes up to 100 chars
ALTER TABLE urls ALTER COLUMN code TYPE VARCHAR(100);

-- Title for display/organization
ALTER TABLE urls ADD COLUMN IF NOT EXISTS title VARCHAR(512);

-- Max visits: auto-disable after N clicks (NULL = unlimited)
ALTER TABLE urls ADD COLUMN IF NOT EXISTS max_visits INTEGER;

-- Redirect type: 301 (permanent), 302 (found), 307 (temp), 308 (permanent redirect)
ALTER TABLE urls ADD COLUMN IF NOT EXISTS redirect_type SMALLINT NOT NULL DEFAULT 302;

-- Crawlable: whether search engines should follow this redirect
ALTER TABLE urls ADD COLUMN IF NOT EXISTS is_crawlable BOOLEAN NOT NULL DEFAULT TRUE;

-- Domain: which domain this short URL is served on (NULL = default)
ALTER TABLE urls ADD COLUMN IF NOT EXISTS domain VARCHAR(255);

-- Index for domain-based lookups
CREATE INDEX IF NOT EXISTS idx_urls_domain ON urls(domain) WHERE domain IS NOT NULL;

-- Domains table for multi-domain support
CREATE TABLE IF NOT EXISTS domains (
    id          BIGSERIAL PRIMARY KEY,
    authority   VARCHAR(255) UNIQUE NOT NULL,
    base_url    VARCHAR(512) NOT NULL,
    is_default  BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
