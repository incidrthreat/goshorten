package shortener

import (
	"context"
	"errors"
	"time"

	pb "github.com/incidrthreat/goshorten/backend/pb"

	"github.com/incidrthreat/goshorten/backend/data"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// AnalyticsFields on CreateServer: AnalyticsStore must be set for analytics RPCs.

// GetVisitSummary returns aggregate visit stats for a short URL.
func (c *CreateServer) GetVisitSummary(ctx context.Context, req *pb.GetVisitSummaryRequest) (*pb.VisitSummaryResponse, error) {
	code := req.GetCode()
	if code == "" {
		return nil, errNoCode
	}

	rec, err := c.Store.Get(code)
	if err != nil {
		return nil, errors.New("code not found")
	}

	var since *time.Time
	if req.GetSince() != nil {
		t := req.GetSince().AsTime()
		since = &t
	}

	summary, err := c.Analytics.GetVisitSummary(rec.ID, since, req.GetExcludeBots())
	if err != nil {
		return nil, err
	}

	return &pb.VisitSummaryResponse{
		Code:           code,
		TotalVisits:    summary.TotalVisits,
		UniqueVisitors: summary.UniqueVisitors,
		BotVisits:      summary.BotVisits,
		HumanVisits:    summary.HumanVisits,
	}, nil
}

// GetVisitsByDate returns daily visit counts for a short URL.
func (c *CreateServer) GetVisitsByDate(ctx context.Context, req *pb.GetVisitsByDateRequest) (*pb.VisitsByDateResponse, error) {
	code := req.GetCode()
	if code == "" {
		return nil, errNoCode
	}

	rec, err := c.Store.Get(code)
	if err != nil {
		return nil, errors.New("code not found")
	}

	since := time.Now().AddDate(0, 0, -30) // default: last 30 days
	if req.GetSince() != nil {
		since = req.GetSince().AsTime()
	}
	until := time.Now()
	if req.GetUntil() != nil {
		until = req.GetUntil().AsTime()
	}

	entries, err := c.Analytics.GetVisitsByDate(rec.ID, since, until, req.GetExcludeBots())
	if err != nil {
		return nil, err
	}

	resp := &pb.VisitsByDateResponse{Code: code}
	for _, e := range entries {
		resp.Entries = append(resp.Entries, &pb.VisitsByDateEntry{
			Date:   e.Date,
			Visits: e.Visits,
		})
	}
	return resp, nil
}

// GetVisitsByField returns visit counts grouped by a field (country, browser, etc.)
func (c *CreateServer) GetVisitsByField(ctx context.Context, req *pb.GetVisitsByFieldRequest) (*pb.VisitsByFieldResponse, error) {
	code := req.GetCode()
	if code == "" {
		return nil, errNoCode
	}

	rec, err := c.Store.Get(code)
	if err != nil {
		return nil, errors.New("code not found")
	}

	var since *time.Time
	if req.GetSince() != nil {
		t := req.GetSince().AsTime()
		since = &t
	}

	entries, err := c.Analytics.GetVisitsByField(rec.ID, req.GetField(), since, req.GetExcludeBots(), int(req.GetLimit()))
	if err != nil {
		return nil, err
	}

	resp := &pb.VisitsByFieldResponse{Code: code, Field: req.GetField()}
	for _, e := range entries {
		resp.Entries = append(resp.Entries, &pb.VisitsByFieldEntry{
			Value:  e.Value,
			Visits: e.Visits,
		})
	}
	return resp, nil
}

// GetRecentVisits returns the most recent visits for a short URL.
func (c *CreateServer) GetRecentVisits(ctx context.Context, req *pb.GetRecentVisitsRequest) (*pb.RecentVisitsResponse, error) {
	code := req.GetCode()
	if code == "" {
		return nil, errNoCode
	}

	rec, err := c.Store.Get(code)
	if err != nil {
		return nil, errors.New("code not found")
	}

	visits, err := c.Analytics.GetRecentVisits(rec.ID, int(req.GetLimit()), req.GetExcludeBots())
	if err != nil {
		return nil, err
	}

	resp := &pb.RecentVisitsResponse{Code: code}
	for _, v := range visits {
		resp.Visits = append(resp.Visits, visitToProto(v))
	}
	return resp, nil
}

// GetOrphanVisits returns recent orphan visits (visits to invalid/expired codes).
func (c *CreateServer) GetOrphanVisits(ctx context.Context, req *pb.GetOrphanVisitsRequest) (*pb.OrphanVisitsResponse, error) {
	visits, err := c.Analytics.GetOrphanVisits(int(req.GetLimit()))
	if err != nil {
		return nil, err
	}

	totalCount, err := c.Analytics.GetOrphanVisitCount(nil)
	if err != nil {
		totalCount = 0
	}

	resp := &pb.OrphanVisitsResponse{TotalCount: totalCount}
	for _, v := range visits {
		resp.Visits = append(resp.Visits, &pb.OrphanVisitEntry{
			Code:      v.Code,
			VisitedAt: timestamppb.New(v.ClickedAt),
			IpAddress: v.IPAddress,
			UserAgent: v.UserAgent,
			Referer:   v.Referer,
			Country:   v.Country,
			City:      v.City,
			IsBot:     v.IsBot,
		})
	}
	return resp, nil
}

func visitToProto(v data.Visit) *pb.VisitEntry {
	return &pb.VisitEntry{
		VisitedAt:  timestamppb.New(v.ClickedAt),
		IpAddress:  v.IPAddress,
		UserAgent:  v.UserAgent,
		Referer:    v.Referer,
		Country:    v.Country,
		City:       v.City,
		DeviceType: v.DeviceType,
		Browser:    v.Browser,
		Os:         v.OS,
		IsBot:      v.IsBot,
	}
}
