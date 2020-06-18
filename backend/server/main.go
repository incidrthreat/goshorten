package main

import (
	"net"
	"os"

	pb "github.com/incidrthreat/goshorten/pb/shortener"

	"github.com/hashicorp/go-hclog"
	"github.com/incidrthreat/goshorten/backend/server/config"
	"github.com/incidrthreat/goshorten/backend/server/shortener"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

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

	pb.RegisterShortenerServer(gs, &shortener.Server{})

	gs.Serve(lis)

}
