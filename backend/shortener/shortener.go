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
func (c *CreateServer) CreateURL(ctx context.Context, req *pb.URL) (*pb.Code, error) {
	url := req.GetLongUrl()
	ttl := req.GetTTL()

	switch ttl {
	case 300, 86400, 172800:
		if url == "" {
			log.Error("Redis Error:", "Error", hclog.Fmt("No URL Reqested"))
			return &pb.Code{}, errors.New("No URL Requested")
		}

		log.Info("CreateURL Req", "Shorten Request", hclog.Fmt("%s", url))

		code, err := c.Store.Save(url, ttl)
		if err != nil {
			log.Error("Redis Save:", "Unable to save", err)
			return &pb.Code{}, errors.New("Unable to store URL")
		}

		resp := &pb.Code{
			Code: code,
		}
		return resp, nil
	default:
		log.Error("TTL Error:", "Error", hclog.Fmt("TTL requested not acceptable"))
		return &pb.Code{}, errors.New("TTL requested not acceptable")
	}

}

// GetURL ...
func (c *CreateServer) GetURL(ctx context.Context, req *pb.Code) (*pb.URL, error) {
	code := req.GetCode()

	if code == "" {
		log.Error("GetURL", "Error", "No Code Reqested")
		return &pb.URL{}, errors.New("No Code Requested")
	}

	fullURL, err := c.Store.Load(code)
	if err != nil {
		log.Error("Redis Load:", "Unable to Load URL", err)
		return &pb.URL{}, errors.New("URL expired or not in storage")
	}

	resp := &pb.URL{
		LongUrl: fullURL,
	}

	return resp, nil

}
