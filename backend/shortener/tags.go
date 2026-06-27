package shortener

import (
	"context"
	"errors"

	pb "github.com/incidrthreat/goshorten/backend/pb"
)

// ListTags returns all tags with URL counts.
func (c *CreateServer) ListTags(ctx context.Context, req *pb.ListTagsRequest) (*pb.ListTagsResponse, error) {
	tags, err := c.Tags.ListTags()
	if err != nil {
		return nil, err
	}

	resp := &pb.ListTagsResponse{}
	for _, t := range tags {
		resp.Tags = append(resp.Tags, &pb.TagInfo{
			Id:       t.ID,
			Name:     t.Name,
			UrlCount: t.URLCount,
		})
	}
	return resp, nil
}

// CreateTag creates a new tag (or returns existing).
func (c *CreateServer) CreateTag(ctx context.Context, req *pb.CreateTagRequest) (*pb.TagInfo, error) {
	name := req.GetName()
	if name == "" {
		return nil, errors.New("tag name is required")
	}

	t, err := c.Tags.CreateTag(name)
	if err != nil {
		return nil, err
	}

	return &pb.TagInfo{
		Id:       t.ID,
		Name:     t.Name,
		UrlCount: t.URLCount,
	}, nil
}

// RenameTag renames an existing tag.
func (c *CreateServer) RenameTag(ctx context.Context, req *pb.RenameTagRequest) (*pb.TagInfo, error) {
	if req.GetOldName() == "" || req.GetNewName() == "" {
		return nil, errors.New("both old_name and new_name are required")
	}

	t, err := c.Tags.RenameTag(req.GetOldName(), req.GetNewName())
	if err != nil {
		return nil, err
	}

	return &pb.TagInfo{
		Id:       t.ID,
		Name:     t.Name,
		UrlCount: t.URLCount,
	}, nil
}

// DeleteTag removes a tag and all its URL associations.
func (c *CreateServer) DeleteTag(ctx context.Context, req *pb.DeleteTagRequest) (*pb.DeleteTagResponse, error) {
	if req.GetName() == "" {
		return nil, errors.New("tag name is required")
	}

	if err := c.Tags.DeleteTag(req.GetName()); err != nil {
		return &pb.DeleteTagResponse{Success: false}, err
	}

	return &pb.DeleteTagResponse{Success: true}, nil
}

// GetTagStats returns aggregated stats for all URLs under a tag.
func (c *CreateServer) GetTagStats(ctx context.Context, req *pb.GetTagStatsRequest) (*pb.TagStatsResponse, error) {
	if req.GetName() == "" {
		return nil, errors.New("tag name is required")
	}

	stats, err := c.Tags.GetTagStats(req.GetName())
	if err != nil {
		return nil, err
	}

	return &pb.TagStatsResponse{
		Tag: &pb.TagInfo{
			Id:       stats.Tag.ID,
			Name:     stats.Tag.Name,
			UrlCount: stats.Tag.URLCount,
		},
		TotalClicks: stats.TotalClicks,
		UniqueUrls:  stats.UniqueURLs,
	}, nil
}
