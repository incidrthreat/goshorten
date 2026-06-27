package data

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Tag represents a tag entity.
type Tag struct {
	ID       int64
	Name     string
	URLCount int64
}

// TagStats holds aggregated analytics for all URLs under a tag.
type TagStats struct {
	Tag         Tag
	TotalClicks int64
	UniqueURLs  int64
}

// TagStore provides tag CRUD and aggregation queries.
type TagStore struct {
	Pool *pgxpool.Pool
}

// ListTags returns all tags with their URL counts.
func (ts *TagStore) ListTags() ([]Tag, error) {
	rows, err := ts.Pool.Query(context.Background(),
		`SELECT t.id, t.name, COUNT(ut.url_id) AS url_count
		 FROM tags t
		 LEFT JOIN url_tags ut ON ut.tag_id = t.id
		 GROUP BY t.id
		 ORDER BY t.name`)
	if err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}
	defer rows.Close()

	var tags []Tag
	for rows.Next() {
		var t Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.URLCount); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, nil
}

// CreateTag creates a new tag. Returns the tag if it already exists.
func (ts *TagStore) CreateTag(name string) (*Tag, error) {
	if name == "" {
		return nil, errors.New("tag name cannot be empty")
	}

	var t Tag
	err := ts.Pool.QueryRow(context.Background(),
		`INSERT INTO tags (name) VALUES ($1)
		 ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
		 RETURNING id, name`, name).Scan(&t.ID, &t.Name)
	if err != nil {
		return nil, fmt.Errorf("create tag: %w", err)
	}

	// Get URL count
	_ = ts.Pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM url_tags WHERE tag_id = $1`, t.ID).Scan(&t.URLCount)

	return &t, nil
}

// RenameTag renames an existing tag.
func (ts *TagStore) RenameTag(oldName, newName string) (*Tag, error) {
	if oldName == "" || newName == "" {
		return nil, errors.New("tag names cannot be empty")
	}

	var t Tag
	err := ts.Pool.QueryRow(context.Background(),
		`UPDATE tags SET name = $1 WHERE name = $2 RETURNING id, name`,
		newName, oldName).Scan(&t.ID, &t.Name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("tag not found")
		}
		if isUniqueViolation(err) {
			return nil, errors.New("a tag with that name already exists")
		}
		return nil, fmt.Errorf("rename tag: %w", err)
	}

	_ = ts.Pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM url_tags WHERE tag_id = $1`, t.ID).Scan(&t.URLCount)

	return &t, nil
}

// DeleteTag removes a tag and all its URL associations.
func (ts *TagStore) DeleteTag(name string) error {
	result, err := ts.Pool.Exec(context.Background(),
		`DELETE FROM tags WHERE name = $1`, name)
	if err != nil {
		return fmt.Errorf("delete tag: %w", err)
	}
	if result.RowsAffected() == 0 {
		return errors.New("tag not found")
	}
	return nil
}

// GetTagStats returns aggregated visit stats for all URLs under a tag.
func (ts *TagStore) GetTagStats(name string) (*TagStats, error) {
	var tagID int64
	var tagName string
	err := ts.Pool.QueryRow(context.Background(),
		`SELECT id, name FROM tags WHERE name = $1`, name).Scan(&tagID, &tagName)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("tag not found")
		}
		return nil, fmt.Errorf("get tag: %w", err)
	}

	var stats TagStats
	stats.Tag = Tag{ID: tagID, Name: tagName}

	err = ts.Pool.QueryRow(context.Background(),
		`SELECT COUNT(DISTINCT ut.url_id), COALESCE(SUM(sub.clicks), 0)
		 FROM url_tags ut
		 LEFT JOIN (
			SELECT url_id, COUNT(*) AS clicks FROM clicks GROUP BY url_id
		 ) sub ON sub.url_id = ut.url_id
		 WHERE ut.tag_id = $1`, tagID).Scan(&stats.UniqueURLs, &stats.TotalClicks)
	if err != nil {
		return nil, fmt.Errorf("tag stats: %w", err)
	}

	stats.Tag.URLCount = stats.UniqueURLs
	return &stats, nil
}
