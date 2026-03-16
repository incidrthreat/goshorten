package data

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// AnalyticsStore provides aggregation queries for visit data.
type AnalyticsStore struct {
	Pool *pgxpool.Pool
}

// GetVisitSummary returns aggregate visit counts for a URL.
func (a *AnalyticsStore) GetVisitSummary(urlID int64, since *time.Time, excludeBots bool) (*VisitSummary, error) {
	where := "url_id = $1"
	args := []interface{}{urlID}
	argIdx := 2

	if since != nil {
		where += fmt.Sprintf(" AND clicked_at >= $%d", argIdx)
		args = append(args, *since)
		argIdx++
	}
	if excludeBots {
		where += " AND is_bot = FALSE"
	}

	var summary VisitSummary
	err := a.Pool.QueryRow(context.Background(), fmt.Sprintf(
		`SELECT
			COUNT(*) AS total_visits,
			COUNT(DISTINCT ip_address) AS unique_visitors,
			COUNT(*) FILTER (WHERE is_bot = TRUE) AS bot_visits,
			COUNT(*) FILTER (WHERE is_bot = FALSE) AS human_visits
		 FROM clicks WHERE %s`, where), args...).
		Scan(&summary.TotalVisits, &summary.UniqueVisitors, &summary.BotVisits, &summary.HumanVisits)
	if err != nil {
		return nil, fmt.Errorf("visit summary: %w", err)
	}
	return &summary, nil
}

// GetVisitsByDate returns daily visit counts for a URL within a date range.
func (a *AnalyticsStore) GetVisitsByDate(urlID int64, since, until time.Time, excludeBots bool) ([]VisitsByDate, error) {
	botFilter := ""
	if excludeBots {
		botFilter = "AND is_bot = FALSE"
	}

	query := fmt.Sprintf(
		`SELECT DATE(clicked_at) AS day, COUNT(*) AS visits
		 FROM clicks
		 WHERE url_id = $1 AND clicked_at >= $2 AND clicked_at < $3 %s
		 GROUP BY day ORDER BY day`, botFilter)

	rows, err := a.Pool.Query(context.Background(), query, urlID, since, until)
	if err != nil {
		return nil, fmt.Errorf("visits by date: %w", err)
	}
	defer rows.Close()

	var result []VisitsByDate
	for rows.Next() {
		var vd VisitsByDate
		var day time.Time
		if err := rows.Scan(&day, &vd.Visits); err != nil {
			return nil, err
		}
		vd.Date = day.Format("2006-01-02")
		result = append(result, vd)
	}
	return result, nil
}

// GetVisitsByField returns visit counts grouped by a column (country, city, browser, os, device_type, referer).
func (a *AnalyticsStore) GetVisitsByField(urlID int64, field string, since *time.Time, excludeBots bool, limit int) ([]VisitsByField, error) {
	// Whitelist field names to prevent SQL injection
	validFields := map[string]string{
		"country":     "country",
		"city":        "city",
		"browser":     "browser",
		"os":          "os",
		"device_type": "device_type",
		"referer":     "referer",
	}
	col, ok := validFields[field]
	if !ok {
		return nil, fmt.Errorf("invalid analytics field: %s", field)
	}

	where := fmt.Sprintf("url_id = $1 AND %s IS NOT NULL AND %s != ''", col, col)
	args := []interface{}{urlID}
	argIdx := 2

	if since != nil {
		where += fmt.Sprintf(" AND clicked_at >= $%d", argIdx)
		args = append(args, *since)
		argIdx++
	}
	if excludeBots {
		where += " AND is_bot = FALSE"
	}

	if limit <= 0 || limit > 100 {
		limit = 20
	}

	query := fmt.Sprintf(
		`SELECT %s AS value, COUNT(*) AS visits
		 FROM clicks WHERE %s
		 GROUP BY %s ORDER BY visits DESC LIMIT %d`,
		col, where, col, limit)

	rows, err := a.Pool.Query(context.Background(), query, args...)
	if err != nil {
		return nil, fmt.Errorf("visits by %s: %w", field, err)
	}
	defer rows.Close()

	var result []VisitsByField
	for rows.Next() {
		var vf VisitsByField
		if err := rows.Scan(&vf.Value, &vf.Visits); err != nil {
			return nil, err
		}
		result = append(result, vf)
	}
	return result, nil
}

// GetRecentVisits returns the most recent visits for a URL.
func (a *AnalyticsStore) GetRecentVisits(urlID int64, limit int, excludeBots bool) ([]Visit, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	botFilter := ""
	if excludeBots {
		botFilter = "AND is_bot = FALSE"
	}

	query := fmt.Sprintf(
		`SELECT url_id, clicked_at,
			COALESCE(host(ip_address)::text, ''), COALESCE(user_agent, ''),
			COALESCE(referer, ''), COALESCE(country, ''), COALESCE(city, ''),
			COALESCE(device_type, ''), COALESCE(browser, ''), COALESCE(os, ''), is_bot
		 FROM clicks
		 WHERE url_id = $1 %s
		 ORDER BY clicked_at DESC LIMIT %d`, botFilter, limit)

	rows, err := a.Pool.Query(context.Background(), query, urlID)
	if err != nil {
		return nil, fmt.Errorf("recent visits: %w", err)
	}
	defer rows.Close()

	var visits []Visit
	for rows.Next() {
		var v Visit
		if err := rows.Scan(&v.URLID, &v.ClickedAt, &v.IPAddress, &v.UserAgent,
			&v.Referer, &v.Country, &v.City, &v.DeviceType, &v.Browser, &v.OS, &v.IsBot); err != nil {
			return nil, err
		}
		visits = append(visits, v)
	}
	return visits, nil
}

// GetOrphanVisitCount returns the total number of orphan visits.
func (a *AnalyticsStore) GetOrphanVisitCount(since *time.Time) (int64, error) {
	where := "1=1"
	args := []interface{}{}
	if since != nil {
		where = "visited_at >= $1"
		args = append(args, *since)
	}

	var count int64
	err := a.Pool.QueryRow(context.Background(),
		fmt.Sprintf("SELECT COUNT(*) FROM orphan_visits WHERE %s", where), args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("orphan count: %w", err)
	}
	return count, nil
}

// GetOrphanVisits returns recent orphan visits.
func (a *AnalyticsStore) GetOrphanVisits(limit int) ([]Visit, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	rows, err := a.Pool.Query(context.Background(),
		`SELECT code, visited_at, COALESCE(host(ip_address)::text, ''),
			COALESCE(user_agent, ''), COALESCE(referer, ''),
			COALESCE(country, ''), COALESCE(city, ''), is_bot
		 FROM orphan_visits
		 ORDER BY visited_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("orphan visits: %w", err)
	}
	defer rows.Close()

	var visits []Visit
	for rows.Next() {
		var v Visit
		if err := rows.Scan(&v.Code, &v.ClickedAt, &v.IPAddress, &v.UserAgent,
			&v.Referer, &v.Country, &v.City, &v.IsBot); err != nil {
			return nil, err
		}
		visits = append(visits, v)
	}
	return visits, nil
}
