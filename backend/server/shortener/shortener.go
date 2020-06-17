package shortener

import (
	"context"

	"github.com/hashicorp/go-hclog"
	pb "github.com/incidrthreat/goshorten/pb/shortener"
)

type Shortener struct {
	long_url string
	log      hclog.Logger
}

func NewURL(lurl string, l hclog.Logger) *Shortener {
	return &Shortener{lurl, l}
}

func (s *Shortener) GetURL(ctx context.Context, ur *pb.URLRequest) (*pb.URLResponse, error) {
	return nil, nil
}
