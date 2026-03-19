package shortener

import (
	"context"
	"errors"

	pb "github.com/incidrthreat/goshorten/backend/pb"

	"github.com/hashicorp/go-hclog"
	"github.com/incidrthreat/goshorten/backend/auth"
	"github.com/incidrthreat/goshorten/backend/data"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	errNoURL            = errors.New("no URL requested")
	errInvalidURL       = errors.New("invalid URL")
	errNoCode           = errors.New("no code requested")
	errSlugTooShort     = errors.New("custom slug must be at least 3 characters")
	errSlugTooLong      = errors.New("custom slug must be at most 100 characters")
	errSlugInvalidChars = errors.New("custom slug may only contain letters, numbers, hyphens, and underscores")
)

// CreateServer holds the backing store and implements the Shortener gRPC service.
type CreateServer struct {
	pb.UnimplementedShortenerServer
	Store     data.URLStore
	Analytics *data.AnalyticsStore
	Tags      *data.TagStore
}

var log = hclog.Default()

// CreateURL creates a new short URL.
func (c *CreateServer) CreateURL(ctx context.Context, req *pb.CreateURLRequest) (*pb.ShortURL, error) {
	longURL, err := NormalizeURL(req.GetLongUrl())
	if err != nil {
		return nil, err
	}

	// Validate custom slug if provided
	if req.GetCustomSlug() != "" {
		if err := ValidateCustomSlug(req.GetCustomSlug()); err != nil {
			return nil, err
		}
	}

	params := data.URLCreateParams{
		LongURL:      longURL,
		TTL:          req.GetTtl(),
		CustomSlug:   req.GetCustomSlug(),
		Title:        req.GetTitle(),
		RedirectType: ValidateRedirectType(req.GetRedirectType()),
		IsCrawlable:  req.GetIsCrawlable(),
		Tags:         req.GetTags(),
	}

	if userID, ok := auth.UserIDFromContext(ctx); ok {
		params.UserID = &userID
	}

	if req.GetMaxVisits() > 0 {
		mv := req.GetMaxVisits()
		params.MaxVisits = &mv
	}

	if req.GetDomain() != "" {
		d := req.GetDomain()
		params.Domain = &d
	}

	rec, err := c.Store.Create(params)
	if err != nil {
		log.Error("CreateURL", "Error", err)
		return nil, err
	}

	return urlRecordToProto(rec), nil
}

// GetURL retrieves a URL by its short code (for redirect).
func (c *CreateServer) GetURL(ctx context.Context, req *pb.GetURLRequest) (*pb.ShortURL, error) {
	code := req.GetCode()
	if code == "" {
		return nil, errNoCode
	}

	hasVisitor := req.GetVisitorIp() != "" || req.GetVisitorUa() != ""

	if hasVisitor {
		// Redirect path: Load() checks expiry/active/max-visits.
		// RecordVisit() records the click with full visitor metadata.
		fullURL, err := c.Store.Load(code)
		if err != nil {
			return nil, errors.New("URL expired or not in storage")
		}
		c.Store.RecordVisit(code, req.GetVisitorIp(), req.GetVisitorUa(), req.GetVisitorReferer())
		rec, err := c.Store.Get(code)
		if err != nil {
			return &pb.ShortURL{LongUrl: fullURL, RedirectType: 302}, nil
		}
		return urlRecordToProto(rec), nil
	}

	// API lookup path (e.g. edit form): use Get() which does not record a click.
	rec, err := c.Store.Get(code)
	if err != nil {
		return nil, errors.New("URL not found")
	}
	return urlRecordToProto(rec), nil
}

// GetStats returns statistics for a short code.
func (c *CreateServer) GetStats(ctx context.Context, req *pb.GetStatsRequest) (*pb.StatsResponse, error) {
	code := req.GetCode()
	if code == "" {
		return nil, errNoCode
	}

	rec, err := c.Store.Get(code)
	if err != nil {
		return nil, errors.New("code expired or not in storage")
	}

	resp := &pb.StatsResponse{
		Code:        rec.Code,
		LongUrl:     rec.LongURL,
		Title:       rec.Title,
		CreatedAt:   timestamppb.New(rec.CreatedAt),
		TotalClicks: rec.TotalClicks,
		IsActive:    rec.IsActive,
		Tags:        rec.Tags,
	}

	if rec.ExpiresAt != nil {
		resp.ExpiresAt = timestamppb.New(*rec.ExpiresAt)
	}
	if rec.MaxVisits != nil {
		resp.MaxVisits = *rec.MaxVisits
	}

	return resp, nil
}

// UpdateURL modifies an existing short URL.
func (c *CreateServer) UpdateURL(ctx context.Context, req *pb.UpdateURLRequest) (*pb.ShortURL, error) {
	code := req.GetCode()
	if code == "" {
		return nil, errNoCode
	}

	// Ownership check: non-admins may only update URLs they created.
	if role := auth.RoleFromContext(ctx); role != "admin" {
		rec, err := c.Store.Get(code)
		if err != nil {
			return nil, status.Error(codes.NotFound, "URL not found")
		}
		userID, ok := auth.UserIDFromContext(ctx)
		if !ok || rec.CreatedByUserID == nil || *rec.CreatedByUserID != userID {
			return nil, status.Error(codes.PermissionDenied, "you do not own this URL")
		}
	}

	params := data.URLUpdateParams{
		Code: code,
	}

	if req.GetLongUrl() != "" {
		normalized, err := NormalizeURL(req.GetLongUrl())
		if err != nil {
			return nil, err
		}
		params.LongURL = &normalized
	}
	if req.GetTitle() != "" {
		t := req.GetTitle()
		params.Title = &t
	}
	if req.GetTtl() != 0 {
		ttl := req.GetTtl()
		params.TTL = &ttl
	}
	if req.GetMaxVisits() != 0 {
		mv := req.GetMaxVisits()
		params.MaxVisits = &mv
	}
	if req.GetRedirectType() != 0 {
		rt := ValidateRedirectType(req.GetRedirectType())
		params.RedirectType = &rt
	}
	if req.GetIsCrawlable() != nil {
		v := req.GetIsCrawlable().GetValue()
		params.IsCrawlable = &v
	}
	if req.GetIsActive() != nil {
		v := req.GetIsActive().GetValue()
		params.IsActive = &v
	}
	if len(req.GetTags()) > 0 {
		params.Tags = req.GetTags()
	}

	rec, err := c.Store.Update(params)
	if err != nil {
		log.Error("UpdateURL", "Error", err)
		return nil, err
	}

	return urlRecordToProto(rec), nil
}

// DeleteURL soft-deletes a short URL.
func (c *CreateServer) DeleteURL(ctx context.Context, req *pb.DeleteURLRequest) (*pb.DeleteURLResponse, error) {
	code := req.GetCode()
	if code == "" {
		return nil, errNoCode
	}

	// Ownership check: non-admins may only delete URLs they created.
	if role := auth.RoleFromContext(ctx); role != "admin" {
		rec, err := c.Store.Get(code)
		if err != nil {
			return nil, status.Error(codes.NotFound, "URL not found")
		}
		userID, ok := auth.UserIDFromContext(ctx)
		if !ok || rec.CreatedByUserID == nil || *rec.CreatedByUserID != userID {
			return nil, status.Error(codes.PermissionDenied, "you do not own this URL")
		}
	}

	if err := c.Store.Delete(code); err != nil {
		log.Error("DeleteURL", "Error", err)
		return &pb.DeleteURLResponse{Success: false}, err
	}

	return &pb.DeleteURLResponse{Success: true}, nil
}

// ListURLs returns a paginated list of short URLs.
func (c *CreateServer) ListURLs(ctx context.Context, req *pb.ListURLsRequest) (*pb.ListURLsResponse, error) {
	params := data.URLListParams{
		Page:     req.GetPage(),
		PageSize: req.GetPageSize(),
		Search:   req.GetSearch(),
		Tag:      req.GetTag(),
		Domain:   req.GetDomain(),
		OrderBy:  req.GetOrderBy(),
		OrderDir: req.GetOrderDir(),
	}

	// Non-admins see only their own URLs.
	if role := auth.RoleFromContext(ctx); role != "admin" {
		if userID, ok := auth.UserIDFromContext(ctx); ok {
			params.UserID = &userID
		}
	}

	result, err := c.Store.List(params)
	if err != nil {
		log.Error("ListURLs", "Error", err)
		return nil, err
	}

	resp := &pb.ListURLsResponse{
		Total:    result.Total,
		Page:     result.Page,
		PageSize: result.PageSize,
	}

	for _, rec := range result.URLs {
		r := rec // avoid loop variable capture
		resp.Urls = append(resp.Urls, urlRecordToProto(&r))
	}

	return resp, nil
}

// GetQRCode generates a QR code PNG for a short URL.
func (c *CreateServer) GetQRCode(ctx context.Context, req *pb.GetQRCodeRequest) (*pb.QRCodeResponse, error) {
	code := req.GetCode()
	if code == "" {
		return nil, errNoCode
	}

	// Verify the code exists
	_, err := c.Store.Get(code)
	if err != nil {
		return nil, errors.New("code not found")
	}

	size := int(req.GetSize())
	if size <= 0 || size > 1000 {
		size = 300
	}

	png, err := GenerateQR(code, size)
	if err != nil {
		return nil, err
	}

	return &pb.QRCodeResponse{
		Image:       png,
		ContentType: "image/png",
	}, nil
}

// PreviewURL returns public link preview info (no auth required).
func (c *CreateServer) PreviewURL(ctx context.Context, req *pb.PreviewURLRequest) (*pb.PreviewURLResponse, error) {
	code := req.GetCode()
	if code == "" {
		return nil, errNoCode
	}

	rec, err := c.Store.Get(code)
	if err != nil {
		return nil, errors.New("code not found or expired")
	}

	resp := &pb.PreviewURLResponse{
		Code:        rec.Code,
		LongUrl:     rec.LongURL,
		Title:       rec.Title,
		CreatedAt:   timestamppb.New(rec.CreatedAt),
		TotalClicks: rec.TotalClicks,
		IsActive:    rec.IsActive,
		Tags:        rec.Tags,
	}
	if rec.Domain != nil {
		resp.Domain = *rec.Domain
	}

	return resp, nil
}

// urlRecordToProto converts a data.URLRecord to a pb.ShortURL.
func urlRecordToProto(rec *data.URLRecord) *pb.ShortURL {
	su := &pb.ShortURL{
		Code:         rec.Code,
		LongUrl:      rec.LongURL,
		Title:        rec.Title,
		CreatedAt:    timestamppb.New(rec.CreatedAt),
		IsActive:     rec.IsActive,
		RedirectType: rec.RedirectType,
		IsCrawlable:  rec.IsCrawlable,
		TotalClicks:  rec.TotalClicks,
		Tags:         rec.Tags,
	}
	if rec.ExpiresAt != nil {
		su.ExpiresAt = timestamppb.New(*rec.ExpiresAt)
	}
	if rec.MaxVisits != nil {
		su.MaxVisits = *rec.MaxVisits
	}
	if rec.Domain != nil {
		su.Domain = *rec.Domain
	}
	return su
}
