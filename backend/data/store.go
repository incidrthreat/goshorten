package data

import "time"

// URLRecord represents a stored shortened URL with all metadata.
type URLRecord struct {
	ID           int64
	Code         string
	LongURL      string
	Title        string
	CreatedAt    time.Time
	ExpiresAt    *time.Time
	IsActive     bool
	MaxVisits    *int32
	RedirectType int32
	IsCrawlable  bool
	Domain       *string
	Tags         []string
	TotalClicks  int64
}

// URLCreateParams holds the input for creating a short URL.
type URLCreateParams struct {
	LongURL      string
	TTL          int64    // seconds, 0 = never
	CustomSlug   string   // user-provided code (empty = auto-generate)
	Title        string
	MaxVisits    *int32   // nil = unlimited
	RedirectType int32    // 301, 302, 307, 308
	IsCrawlable  bool
	Domain       *string
	Tags         []string
}

// URLUpdateParams holds the input for updating a short URL.
type URLUpdateParams struct {
	Code         string
	LongURL      *string  // nil = no change
	Title        *string  // nil = no change
	TTL          *int64   // nil = no change, -1 = never expire
	MaxVisits    *int32   // nil = no change, -1 = unlimited
	RedirectType *int32   // nil = no change
	IsCrawlable  *bool    // nil = no change
	IsActive     *bool    // nil = no change
	Tags         []string // nil = no change, empty = clear all
}

// URLListParams holds the input for listing/filtering short URLs.
type URLListParams struct {
	Page     int32
	PageSize int32
	Search   string
	Tag      string
	Domain   string
	OrderBy  string // "created_at", "clicks", "code"
	OrderDir string // "asc", "desc"
}

// URLListResult holds a paginated list of URL records.
type URLListResult struct {
	URLs     []URLRecord
	Total    int32
	Page     int32
	PageSize int32
}

// URLStore defines the contract for URL persistence.
type URLStore interface {
	// Legacy methods (kept for Phase 1 compatibility)
	Save(url string, ttl int64, stats string) (string, error)
	Load(code string) (string, error)
	Stats(code string) (string, error)

	// Phase 2 methods
	Create(params URLCreateParams) (*URLRecord, error)
	Get(code string) (*URLRecord, error)
	Update(params URLUpdateParams) (*URLRecord, error)
	Delete(code string) error
	List(params URLListParams) (*URLListResult, error)

	// Phase 5: visit logging
	SetVisitLogger(vl *VisitLogger)
	RecordVisit(code string, ipAddress, userAgent, referer string)
}
