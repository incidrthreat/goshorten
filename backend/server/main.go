package main

import (
	"context"
	"net"
	"os"

	pb "github.com/incidrthreat/goshorten/pb/shortener"

	"github.com/hashicorp/go-hclog"
	"github.com/incidrthreat/goshorten/backend/server/config"
	"github.com/incidrthreat/goshorten/backend/server/data"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type server struct{}

var log = hclog.Default()

func (*server) GetURL(ctx context.Context, req *pb.URLRequest) (*pb.URLResponse, error) {
	//longURL := req.Url

	//log.Info("Request made:", longURL)

	code := data.GenCode(6)

	resp := &pb.URLResponse{
		Shortened: code,
	}
	return resp, nil
}

func main() {

	log := hclog.Default()
	conf, err := config.ConfigFromFile("config.json")
	if err != nil {
		log.Error("Problem with Json file", "error", err)
		os.Exit(1)
	}

	lis, err := net.Listen("tcp", conf.GRPCHost)
	if err != nil {
		log.Error("Unable to create listener", "error", err)
		os.Exit(1)
	}

	log.Info("Server staring", "addr:port", hclog.Fmt("listening on %v ...", conf.GRPCHost))

	gs := grpc.NewServer()
	reflection.Register(gs)

	pb.RegisterShortenerServer(gs, &server{})

	gs.Serve(lis)

}
