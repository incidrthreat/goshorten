package shortener

import (
	"context"
	"errors"

	pb "github.com/incidrthreat/goshorten/backend/pb"

	"github.com/hashicorp/go-hclog"
	"github.com/incidrthreat/goshorten/backend/data"
)

// CreateServer holds the redis store data
type CreateServer struct {
	Store data.Redis
}

var log = hclog.Default()

// CreateURL ...
func (c *CreateServer) CreateURL(ctx context.Context, req *pb.ShortURLReq) (*pb.ShortURLResp, error) {
	url := req.GetLongUrl()

	if url == "" {
		log.Error("No URL Reqested")
		return &pb.ShortURLResp{}, errors.New("No URL Requested")
	}

	log.Info("CreateURL Req", "Shorten Request", hclog.Fmt("%s", url))

	code, err := c.Store.Save(url)
	if err != nil {
		log.Error("Redis Save:", "Unable to save", err)
		return &pb.ShortURLResp{}, errors.New("Unable to store URL")
	}

	resp := &pb.ShortURLResp{
		ShortUrl: code,
	}
	return resp, nil

}

// GetURL ...
func (c *CreateServer) GetURL(ctx context.Context, req *pb.URLReq) (*pb.URLResp, error) {
	code := req.GetUrlCode()
	if code == "" {
		log.Error("GetURL", "Error", "No Code Reqested")
		return &pb.URLResp{}, errors.New("No Code Requested")
	}

	fullURL, err := c.Store.Load(code)
	if err != nil {
		log.Error("Redis Load:", "Unable to Load URL", err)
		return &pb.URLResp{}, errors.New("URL expired or not in storage")
	}

	resp := &pb.URLResp{
		RedirectUrl: fullURL,
	}

	return resp, nil

}
