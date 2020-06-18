package shortener

import (
	"context"
	"fmt"

	pb "github.com/incidrthreat/goshorten/pb/shortener"

	"github.com/hashicorp/go-hclog"
	"github.com/incidrthreat/goshorten/backend/server/data"
)

type Server struct{}

var log = hclog.Default()

func (*Server) GetURL(ctx context.Context, req *pb.URLRequest) (*pb.URLResponse, error) {
	if req.Url == "" {
		log.Error("No URL Reqested")
	} else {
		fmt.Printf("URL: %s", req.Url)
		code := data.GenCode(6)

		resp := &pb.URLResponse{
			Shortened: "bit.ly/" + code,
		}
		return resp, nil
	}

	return &pb.URLResponse{}, nil
}
