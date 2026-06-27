package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore implements URLStore backed by Postgres.
type PostgresStore struct {
	Pool        *pgxpool.Pool
	CharFloor   int
	visitLogger *VisitLogger
}

// NewPostgresStore creates a connection pool and returns a ready store.
func NewPostgresStore(connString string, charFloor int) (*PostgresStore, error) {
	pool, err := pgxpool.New(context.Background(), connString)
	if err != nil {
		return nil, fmt.Errorf("unable to create pg pool: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("unable to ping postgres: %w", err)
	}

	log.Info("Postgres Server", "Connection", "Online")
	return &PostgresStore{Pool: pool, CharFloor: charFloor}, nil
}

// --- Legacy Methods (Phase 1 backward compat) ---

// Save inserts a URL into Postgres, generates a unique code, and returns it.
func (p *PostgresStore) Save(url string, ttl int64, stats string) (string, error) {
	rec, err := p.Create(URLCreateParams{
		LongURL:      url,
		TTL:          ttl,
		RedirectType: 302,
		IsCrawlable:  true,
	})
	if err != nil {
		return "", err
	}
	return rec.Code, nil
}

// Load retrieves the original URL for a code and records a click.
func (p *PostgresStore) Load(code string) (string, error) {
	var longURL string
	var urlID int64
	var maxVisits *int32

	err := p.Pool.QueryRow(context.Background(),
		`SELECT id, long_url, max_visits FROM urls
		 WHERE code = $1 AND is_active = TRUE
		 AND (expires_at IS NULL OR expires_at > NOW())`,
		code,
	).Scan(&urlID, &longURL, &maxVisits)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", errors.New("Code not found")
		}
		return "", fmt.Errorf("load url: %w", err)
	}

	// Check max visits
	if maxVisits != nil {
		var clickCount int64
		_ = p.Pool.QueryRow(context.Background(),
			`SELECT COUNT(*) FROM clicks WHERE url_id = $1`, urlID,
		).Scan(&clickCount)
		if clickCount >= int64(*maxVisits) {
			// Auto-disable
			_, _ = p.Pool.Exec(context.Background(),
				`UPDATE urls SET is_active = FALSE WHERE id = $1`, urlID)
			return "", errors.New("Code not found")
		}
	}

	log.Info("Postgres Load", "URL retrieved", longURL)
	return longURL, nil
}

// SetVisitLogger attaches a VisitLogger for async visit recording.
func (p *PostgresStore) SetVisitLogger(vl *VisitLogger) {
	p.visitLogger = vl
}

// RecordVisit records a visit with full metadata for the given code.
func (p *PostgresStore) RecordVisit(code string, ipAddress, userAgent, referer string) {
	// Look up URL ID
	var urlID int64
	err := p.Pool.QueryRow(context.Background(),
		`SELECT id FROM urls WHERE code = $1`, code).Scan(&urlID)
	if err != nil {
		if p.visitLogger != nil {
			p.visitLogger.LogOrphanVisit(Visit{
				Code:      code,
				IPAddress: ipAddress,
				UserAgent: userAgent,
				Referer:   referer,
			})
		}
		return
	}

	if p.visitLogger != nil {
		p.visitLogger.LogVisit(Visit{
			URLID:     urlID,
			IPAddress: ipAddress,
			UserAgent: userAgent,
			Referer:   referer,
		})
	} else {
		go func() {
			_, err := p.Pool.Exec(context.Background(),
				`INSERT INTO clicks (url_id, clicked_at) VALUES ($1, NOW())`, urlID)
			if err != nil {
				log.Error("Postgres Click", "Error recording click", err)
			}
		}()
	}
}

// Stats returns JSON stats for a given code (legacy format for frontend compat).
func (p *PostgresStore) Stats(code string) (string, error) {
	var longURL string
	var createdAt time.Time
	var clicks int64
	var lastAccessed *time.Time

	err := p.Pool.QueryRow(context.Background(),
		`SELECT u.long_url, u.created_at,
			COUNT(c.id) AS clicks,
			MAX(c.clicked_at) AS last_accessed
		 FROM urls u
		 LEFT JOIN clicks c ON c.url_id = u.id
		 WHERE u.code = $1 AND u.is_active = TRUE
		 GROUP BY u.id`,
		code,
	).Scan(&longURL, &createdAt, &clicks, &lastAccessed)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", errors.New("Code not found")
		}
		return "", fmt.Errorf("stats query: %w", err)
	}

	const timeFormat = "Mon, 02 Jan 2006 15:04:05 MST"
	statsMap := map[string]string{
		"code":          code,
		"url":           longURL,
		"created_at":    createdAt.Format(timeFormat),
		"clicks":        fmt.Sprintf("%d", clicks),
		"last_accessed": "",
	}
	if lastAccessed != nil {
		statsMap["last_accessed"] = lastAccessed.Format(timeFormat)
	}

	statsJSON, err := json.Marshal(statsMap)
	if err != nil {
		return "", fmt.Errorf("marshal stats: %w", err)
	}
	return string(statsJSON), nil
}

// --- Phase 2 Methods ---

// Create inserts a new short URL with all metadata.
func (p *PostgresStore) Create(params URLCreateParams) (*URLRecord, error) {
	var expiresAt *time.Time
	if params.TTL > 0 {
		t := time.Now().Add(time.Duration(params.TTL) * time.Second)
		expiresAt = &t
	}

	redirectType := params.RedirectType
	if redirectType == 0 {
		redirectType = 302
	}

	// If custom slug provided, use it directly.
	if params.CustomSlug != "" {
		rec, err := p.insertURL(params.CustomSlug, params, expiresAt, redirectType)
		if err != nil && isUniqueViolation(err) {
			// Give a clear message. If the conflict is an inactive URL (e.g. hit
			// max-visits) tell the user they can re-enable it instead.
			if existing, _ := p.Get(params.CustomSlug); existing != nil && !existing.IsActive {
				return nil, fmt.Errorf("slug %q already exists but is deactivated — edit the existing URL to re-enable it, or choose a different slug", params.CustomSlug)
			}
			return nil, fmt.Errorf("slug %q is already in use", params.CustomSlug)
		}
		return rec, err
	}

	// Auto-generate code with collision retry.
	codeLen := p.CharFloor
	for attempts := 0; attempts < 3; attempts++ {
		code := GenCode(codeLen)
		rec, err := p.insertURL(code, params, expiresAt, redirectType)
		if err != nil {
			if isUniqueViolation(err) {
				log.Warn("Postgres Warning", "Key Collision", fmt.Sprintf("Collision on code: %s", code))
				codeLen++
				continue
			}
			return nil, err
		}
		return rec, nil
	}
	return nil, errors.New("3 code collisions detected, try again later")
}

func (p *PostgresStore) insertURL(code string, params URLCreateParams, expiresAt *time.Time, redirectType int32) (*URLRecord, error) {
	rec := &URLRecord{}
	err := p.Pool.QueryRow(context.Background(),
		`INSERT INTO urls (code, long_url, title, expires_at, max_visits, redirect_type, is_crawlable, domain, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING id, code, long_url, title, created_at, expires_at, is_active, max_visits, redirect_type, is_crawlable, domain, created_by`,
		code, params.LongURL, nullIfEmpty(params.Title), expiresAt,
		params.MaxVisits, redirectType, params.IsCrawlable, params.Domain, params.UserID,
	).Scan(&rec.ID, &rec.Code, &rec.LongURL, &rec.Title, &rec.CreatedAt, &rec.ExpiresAt,
		&rec.IsActive, &rec.MaxVisits, &rec.RedirectType, &rec.IsCrawlable, &rec.Domain,
		&rec.CreatedByUserID)

	if err != nil {
		return nil, fmt.Errorf("insert url: %w", err)
	}

	// Associate tags
	if len(params.Tags) > 0 {
		if err := p.setTags(rec.ID, params.Tags); err != nil {
			log.Warn("Postgres Tags", "Failed to set tags", err)
		}
		rec.Tags = params.Tags
	}

	log.Info("Postgres Create", "Code stored", fmt.Sprintf("Code: %s | URL: %s", rec.Code, rec.LongURL))
	return rec, nil
}

// Get retrieves a URL record by code.
func (p *PostgresStore) Get(code string) (*URLRecord, error) {
	rec := &URLRecord{}
	err := p.Pool.QueryRow(context.Background(),
		`SELECT u.id, u.code, u.long_url, COALESCE(u.title, ''), u.created_at, u.expires_at,
			u.is_active, u.max_visits, u.redirect_type, u.is_crawlable, u.domain,
			COUNT(c.id) AS total_clicks,
			u.created_by, COALESCE(usr.email, '')
		 FROM urls u
		 LEFT JOIN clicks c ON c.url_id = u.id
		 LEFT JOIN users usr ON usr.id = u.created_by
		 WHERE u.code = $1
		 GROUP BY u.id, usr.email`,
		code,
	).Scan(&rec.ID, &rec.Code, &rec.LongURL, &rec.Title, &rec.CreatedAt, &rec.ExpiresAt,
		&rec.IsActive, &rec.MaxVisits, &rec.RedirectType, &rec.IsCrawlable, &rec.Domain,
		&rec.TotalClicks, &rec.CreatedByUserID, &rec.CreatedByEmail)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("Code not found")
		}
		return nil, fmt.Errorf("get url: %w", err)
	}

	tags, _ := p.getTags(rec.ID)
	rec.Tags = tags
	return rec, nil
}

// Update modifies an existing short URL.
func (p *PostgresStore) Update(params URLUpdateParams) (*URLRecord, error) {
	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	if params.LongURL != nil {
		setClauses = append(setClauses, fmt.Sprintf("long_url = $%d", argIdx))
		args = append(args, *params.LongURL)
		argIdx++
	}
	if params.Title != nil {
		setClauses = append(setClauses, fmt.Sprintf("title = $%d", argIdx))
		args = append(args, nullIfEmpty(*params.Title))
		argIdx++
	}
	if params.TTL != nil {
		if *params.TTL == -1 {
			setClauses = append(setClauses, fmt.Sprintf("expires_at = $%d", argIdx))
			args = append(args, nil)
		} else {
			setClauses = append(setClauses, fmt.Sprintf("expires_at = $%d", argIdx))
			t := time.Now().Add(time.Duration(*params.TTL) * time.Second)
			args = append(args, t)
		}
		argIdx++
	}
	if params.MaxVisits != nil {
		if *params.MaxVisits == -1 {
			setClauses = append(setClauses, fmt.Sprintf("max_visits = $%d", argIdx))
			args = append(args, nil)
		} else {
			setClauses = append(setClauses, fmt.Sprintf("max_visits = $%d", argIdx))
			args = append(args, *params.MaxVisits)
		}
		argIdx++
	}
	if params.RedirectType != nil {
		setClauses = append(setClauses, fmt.Sprintf("redirect_type = $%d", argIdx))
		args = append(args, *params.RedirectType)
		argIdx++
	}
	if params.IsCrawlable != nil {
		setClauses = append(setClauses, fmt.Sprintf("is_crawlable = $%d", argIdx))
		args = append(args, *params.IsCrawlable)
		argIdx++
	}
	if params.IsActive != nil {
		setClauses = append(setClauses, fmt.Sprintf("is_active = $%d", argIdx))
		args = append(args, *params.IsActive)
		argIdx++
	}
	if params.AssignedUserID != nil {
		setClauses = append(setClauses, fmt.Sprintf("created_by = $%d", argIdx))
		if *params.AssignedUserID == 0 {
			args = append(args, nil) // clear owner
		} else {
			args = append(args, *params.AssignedUserID)
		}
		argIdx++
	}

	if len(setClauses) == 0 && params.Tags == nil {
		return p.Get(params.Code)
	}

	if len(setClauses) > 0 {
		query := fmt.Sprintf("UPDATE urls SET %s WHERE code = $%d",
			strings.Join(setClauses, ", "), argIdx)
		args = append(args, params.Code)

		result, err := p.Pool.Exec(context.Background(), query, args...)
		if err != nil {
			return nil, fmt.Errorf("update url: %w", err)
		}
		if result.RowsAffected() == 0 {
			return nil, errors.New("Code not found")
		}
	}

	// Update tags if provided
	if params.Tags != nil {
		var urlID int64
		err := p.Pool.QueryRow(context.Background(),
			`SELECT id FROM urls WHERE code = $1`, params.Code).Scan(&urlID)
		if err != nil {
			return nil, fmt.Errorf("update tags: %w", err)
		}
		if err := p.setTags(urlID, params.Tags); err != nil {
			return nil, fmt.Errorf("update tags: %w", err)
		}
	}

	return p.Get(params.Code)
}

// Delete permanently removes a URL and its associated clicks/tags (via CASCADE).
func (p *PostgresStore) Delete(code string) error {
	result, err := p.Pool.Exec(context.Background(),
		`DELETE FROM urls WHERE code = $1`, code)
	if err != nil {
		return fmt.Errorf("delete url: %w", err)
	}
	if result.RowsAffected() == 0 {
		return errors.New("Code not found")
	}
	return nil
}

// List returns a paginated, filterable list of URLs.
func (p *PostgresStore) List(params URLListParams) (*URLListResult, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = 20
	}
	if params.OrderBy == "" {
		params.OrderBy = "created_at"
	}
	if params.OrderDir == "" {
		params.OrderDir = "desc"
	}

	// Validate order fields to prevent SQL injection.
	validOrderBy := map[string]string{
		"created_at": "u.created_at",
		"clicks":     "total_clicks",
		"code":       "u.code",
	}
	orderCol, ok := validOrderBy[params.OrderBy]
	if !ok {
		orderCol = "u.created_at"
	}
	orderDir := "DESC"
	if strings.ToLower(params.OrderDir) == "asc" {
		orderDir = "ASC"
	}

	whereClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	if params.Search != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("(u.code ILIKE $%d OR u.long_url ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+params.Search+"%")
		argIdx++
	}
	if params.Tag != "" {
		whereClauses = append(whereClauses, fmt.Sprintf(
			"EXISTS (SELECT 1 FROM url_tags ut JOIN tags t ON t.id = ut.tag_id WHERE ut.url_id = u.id AND t.name = $%d)", argIdx))
		args = append(args, params.Tag)
		argIdx++
	}
	if params.Domain != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("u.domain = $%d", argIdx))
		args = append(args, params.Domain)
		argIdx++
	}
	if params.UserID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("u.created_by = $%d", argIdx))
		args = append(args, *params.UserID)
		argIdx++
	}

	var whereSQL string
	if len(whereClauses) > 0 {
		whereSQL = strings.Join(whereClauses, " AND ")
	} else {
		whereSQL = "TRUE"
	}

	// Count query
	var total int32
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM urls u WHERE %s", whereSQL)
	if err := p.Pool.QueryRow(context.Background(), countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count urls: %w", err)
	}

	// Data query
	offset := (params.Page - 1) * params.PageSize
	dataQuery := fmt.Sprintf(
		`SELECT u.id, u.code, u.long_url, COALESCE(u.title, ''), u.created_at, u.expires_at,
			u.is_active, u.max_visits, u.redirect_type, u.is_crawlable, u.domain,
			COUNT(c.id) AS total_clicks,
			u.created_by, COALESCE(usr.email, '')
		 FROM urls u
		 LEFT JOIN clicks c ON c.url_id = u.id
		 LEFT JOIN users usr ON usr.id = u.created_by
		 WHERE %s
		 GROUP BY u.id, usr.email
		 ORDER BY %s %s
		 LIMIT $%d OFFSET $%d`,
		whereSQL, orderCol, orderDir, argIdx, argIdx+1)
	args = append(args, params.PageSize, offset)

	rows, err := p.Pool.Query(context.Background(), dataQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("list urls: %w", err)
	}
	defer rows.Close()

	var urls []URLRecord
	for rows.Next() {
		var rec URLRecord
		if err := rows.Scan(&rec.ID, &rec.Code, &rec.LongURL, &rec.Title, &rec.CreatedAt, &rec.ExpiresAt,
			&rec.IsActive, &rec.MaxVisits, &rec.RedirectType, &rec.IsCrawlable, &rec.Domain,
			&rec.TotalClicks, &rec.CreatedByUserID, &rec.CreatedByEmail); err != nil {
			return nil, fmt.Errorf("scan url: %w", err)
		}
		tags, _ := p.getTags(rec.ID)
		rec.Tags = tags
		urls = append(urls, rec)
	}

	return &URLListResult{
		URLs:     urls,
		Total:    total,
		Page:     params.Page,
		PageSize: params.PageSize,
	}, nil
}

// --- Tag helpers ---

func (p *PostgresStore) setTags(urlID int64, tags []string) error {
	// Clear existing tags
	_, err := p.Pool.Exec(context.Background(),
		`DELETE FROM url_tags WHERE url_id = $1`, urlID)
	if err != nil {
		return err
	}

	for _, tagName := range tags {
		tagName = strings.TrimSpace(tagName)
		if tagName == "" {
			continue
		}
		// Upsert tag
		var tagID int64
		err := p.Pool.QueryRow(context.Background(),
			`INSERT INTO tags (name) VALUES ($1) ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name RETURNING id`,
			tagName).Scan(&tagID)
		if err != nil {
			return err
		}
		// Link tag to URL
		_, err = p.Pool.Exec(context.Background(),
			`INSERT INTO url_tags (url_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			urlID, tagID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *PostgresStore) getTags(urlID int64) ([]string, error) {
	rows, err := p.Pool.Query(context.Background(),
		`SELECT t.name FROM tags t JOIN url_tags ut ON ut.tag_id = t.id WHERE ut.url_id = $1 ORDER BY t.name`,
		urlID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tags = append(tags, name)
	}
	return tags, nil
}

// --- Helpers ---

func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// isUniqueViolation checks if the error is a Postgres unique constraint violation (SQLSTATE 23505).
func isUniqueViolation(err error) bool {
	var pgErr interface{ SQLState() string }
	if errors.As(err, &pgErr) {
		return pgErr.SQLState() == "23505"
	}
	return false
}
