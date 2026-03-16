package data

import (
	"time"

	"github.com/go-redis/redis"
)

// CachedStore implements URLStore by combining a primary Postgres store
// with a Redis cache for fast redirect lookups.
type CachedStore struct {
	Primary URLStore      // Postgres (source of truth)
	Cache   *redis.Client // Redis (redirect cache)
}

// --- Legacy Methods ---

// Save writes to Postgres first, then populates the Redis cache (write-through).
func (c *CachedStore) Save(url string, ttl int64, stats string) (string, error) {
	code, err := c.Primary.Save(url, ttl, stats)
	if err != nil {
		return "", err
	}

	cacheTTL := time.Duration(ttl) * time.Second
	if err := c.Cache.Set(code, url, cacheTTL).Err(); err != nil {
		log.Warn("Redis Cache", "Failed to cache URL", err)
	}

	return code, nil
}

// Load checks Redis first (cache hit = fast path), falls back to Postgres on miss.
func (c *CachedStore) Load(code string) (string, error) {
	url, err := c.Cache.Get(code).Result()
	if err == nil {
		log.Info("Cache Hit", "code", code)
		go func() {
			_, _ = c.Primary.Load(code)
		}()
		return url, nil
	}

	log.Info("Cache Miss", "code", code)
	url, err = c.Primary.Load(code)
	if err != nil {
		return "", err
	}

	if err := c.Cache.Set(code, url, 24*time.Hour).Err(); err != nil {
		log.Warn("Redis Cache", "Failed to backfill cache", err)
	}

	return url, nil
}

// Stats always queries Postgres directly (source of truth for analytics).
func (c *CachedStore) Stats(code string) (string, error) {
	return c.Primary.Stats(code)
}

// --- Phase 2 Methods ---

// Create writes to Postgres, then caches the redirect mapping.
func (c *CachedStore) Create(params URLCreateParams) (*URLRecord, error) {
	rec, err := c.Primary.Create(params)
	if err != nil {
		return nil, err
	}

	// Write-through cache
	var cacheTTL time.Duration
	if rec.ExpiresAt != nil {
		cacheTTL = time.Until(*rec.ExpiresAt)
	} else {
		cacheTTL = 24 * time.Hour
	}
	if err := c.Cache.Set(rec.Code, rec.LongURL, cacheTTL).Err(); err != nil {
		log.Warn("Redis Cache", "Failed to cache URL", err)
	}

	return rec, nil
}

// Get retrieves a URL record (always from Postgres for full metadata).
func (c *CachedStore) Get(code string) (*URLRecord, error) {
	return c.Primary.Get(code)
}

// Update modifies a URL and invalidates the cache.
func (c *CachedStore) Update(params URLUpdateParams) (*URLRecord, error) {
	rec, err := c.Primary.Update(params)
	if err != nil {
		return nil, err
	}

	// Invalidate and re-cache with updated URL
	c.invalidate(params.Code)
	if rec.IsActive {
		var cacheTTL time.Duration
		if rec.ExpiresAt != nil {
			cacheTTL = time.Until(*rec.ExpiresAt)
		} else {
			cacheTTL = 24 * time.Hour
		}
		if err := c.Cache.Set(rec.Code, rec.LongURL, cacheTTL).Err(); err != nil {
			log.Warn("Redis Cache", "Failed to re-cache URL", err)
		}
	}

	return rec, nil
}

// Delete soft-deletes in Postgres and removes from cache.
func (c *CachedStore) Delete(code string) error {
	if err := c.Primary.Delete(code); err != nil {
		return err
	}
	c.invalidate(code)
	return nil
}

// List always queries Postgres directly.
func (c *CachedStore) List(params URLListParams) (*URLListResult, error) {
	return c.Primary.List(params)
}

// SetVisitLogger delegates to the primary store.
func (c *CachedStore) SetVisitLogger(vl *VisitLogger) {
	c.Primary.SetVisitLogger(vl)
}

// RecordVisit delegates to the primary store.
func (c *CachedStore) RecordVisit(code string, ipAddress, userAgent, referer string) {
	c.Primary.RecordVisit(code, ipAddress, userAgent, referer)
}

func (c *CachedStore) invalidate(code string) {
	if err := c.Cache.Del(code).Err(); err != nil {
		log.Warn("Redis Cache", "Failed to invalidate", err)
	}
}
