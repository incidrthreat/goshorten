DROP TABLE IF EXISTS orphan_visits;

DROP INDEX IF EXISTS idx_clicks_url_date;
DROP INDEX IF EXISTS idx_clicks_ip_address;
DROP INDEX IF EXISTS idx_clicks_referer;
DROP INDEX IF EXISTS idx_clicks_is_bot;
DROP INDEX IF EXISTS idx_clicks_os;
DROP INDEX IF EXISTS idx_clicks_browser;
DROP INDEX IF EXISTS idx_clicks_country;

ALTER TABLE clicks
    DROP COLUMN IF EXISTS country,
    DROP COLUMN IF EXISTS city,
    DROP COLUMN IF EXISTS device_type,
    DROP COLUMN IF EXISTS browser,
    DROP COLUMN IF EXISTS os,
    DROP COLUMN IF EXISTS is_bot;
