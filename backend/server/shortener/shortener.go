package shortener

import (
	"context"
	"fmt"

	pb "github.com/incidrthreat/goshorten/backend/server/pb"

	"github.com/hashicorp/go-hclog"
	"github.com/incidrthreat/goshorten/backend/server/data"
)

type Server struct{}

var log = hclog.Default()

func (*Server) CreateURL(ctx context.Context, req *pb.ShortURLReq) (*pb.ShortURLResp, error) {
	if req.LongUrl == "" {
		log.Error("No URL Reqested")
	} else {
		fmt.Printf("URL: %s", req.LongUrl)
		code := data.GenCode(6)

		resp := &pb.ShortURLResp{
			ShortUrl: "bit.ly/" + code,
		}
		return resp, nil
	}

	return &pb.ShortURLResp{}, nil
}
