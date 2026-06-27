package data

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Visit represents a single click event with full metadata.
type Visit struct {
	URLID      int64
	Code       string // used for orphan visits (URLID=0)
	ClickedAt  time.Time
	IPAddress  string
	UserAgent  string
	Referer    string
	Country    string
	City       string
	DeviceType string
	Browser    string
	OS         string
	IsBot      bool
}

// VisitSummary holds aggregated visit data for a URL.
type VisitSummary struct {
	TotalVisits    int64
	UniqueVisitors int64
	BotVisits      int64
	HumanVisits    int64
}

// VisitsByDate holds visit counts per day.
type VisitsByDate struct {
	Date   string `json:"date"`
	Visits int64  `json:"visits"`
}

// VisitsByField holds visit counts per a grouped field (referrer, country, browser, etc.)
type VisitsByField struct {
	Value  string `json:"value"`
	Visits int64  `json:"visits"`
}

// VisitLogger handles async visit recording via a buffered channel.
type VisitLogger struct {
	pool    *pgxpool.Pool
	visitCh chan Visit
	done    chan struct{}
	log     hclog.Logger
}

// NewVisitLogger creates a visit logger with a buffered channel and starts workers.
func NewVisitLogger(pool *pgxpool.Pool, bufSize int, workers int) *VisitLogger {
	if bufSize <= 0 {
		bufSize = 4096
	}
	if workers <= 0 {
		workers = 2
	}

	vl := &VisitLogger{
		pool:    pool,
		visitCh: make(chan Visit, bufSize),
		done:    make(chan struct{}),
		log:     hclog.Default(),
	}

	for i := 0; i < workers; i++ {
		go vl.worker(i)
	}

	vl.log.Info("VisitLogger", "started", fmt.Sprintf("%d workers, buffer=%d", workers, bufSize))
	return vl
}

// LogVisit enqueues a visit for async recording. Non-blocking: drops if buffer full.
func (vl *VisitLogger) LogVisit(v Visit) {
	if v.ClickedAt.IsZero() {
		v.ClickedAt = time.Now()
	}

	// Parse user-agent
	if v.UserAgent != "" {
		v.DeviceType, v.Browser, v.OS = ParseUserAgent(v.UserAgent)
		v.IsBot = IsBot(v.UserAgent)
	}

	select {
	case vl.visitCh <- v:
	default:
		vl.log.Warn("VisitLogger", "dropped visit (buffer full)", v.Code)
	}
}

// LogOrphanVisit records a visit to a code that doesn't exist or is inactive.
func (vl *VisitLogger) LogOrphanVisit(v Visit) {
	if v.ClickedAt.IsZero() {
		v.ClickedAt = time.Now()
	}
	if v.UserAgent != "" {
		v.IsBot = IsBot(v.UserAgent)
	}

	// Orphan visits go directly (they're less frequent)
	go func() {
		_, err := vl.pool.Exec(context.Background(),
			`INSERT INTO orphan_visits (code, visited_at, ip_address, user_agent, referer, country, city, is_bot)
			 VALUES ($1, $2, $3::inet, $4, $5, $6, $7, $8)`,
			v.Code, v.ClickedAt, nullIfEmptyInet(v.IPAddress), v.UserAgent,
			nullIfEmpty(v.Referer), nullIfEmpty(v.Country), nullIfEmpty(v.City), v.IsBot)
		if err != nil {
			vl.log.Error("VisitLogger", "orphan insert error", err)
		}
	}()
}

// Close stops the visit logger gracefully.
func (vl *VisitLogger) Close() {
	close(vl.visitCh)
	<-vl.done
}

func (vl *VisitLogger) worker(id int) {
	for v := range vl.visitCh {
		if v.URLID > 0 {
			vl.insertClick(v)
		}
	}
	// Signal done only from last worker (simplified: each worker signals)
	select {
	case vl.done <- struct{}{}:
	default:
	}
}

func (vl *VisitLogger) insertClick(v Visit) {
	_, err := vl.pool.Exec(context.Background(),
		`INSERT INTO clicks (url_id, clicked_at, ip_address, user_agent, referer, country, city, device_type, browser, os, is_bot)
		 VALUES ($1, $2, $3::inet, $4, $5, $6, $7, $8, $9, $10, $11)`,
		v.URLID, v.ClickedAt, nullIfEmptyInet(v.IPAddress), nullIfEmpty(v.UserAgent),
		nullIfEmpty(v.Referer), nullIfEmpty(v.Country), nullIfEmpty(v.City),
		nullIfEmpty(v.DeviceType), nullIfEmpty(v.Browser), nullIfEmpty(v.OS), v.IsBot)
	if err != nil {
		vl.log.Error("VisitLogger", "click insert error", err)
	}
}

func nullIfEmptyInet(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}
