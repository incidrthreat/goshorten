package main

import (
	"net"
	"os"

	"github.com/go-redis/redis"
	"github.com/incidrthreat/goshorten/backend/data"
	pb "github.com/incidrthreat/goshorten/backend/pb"

	"github.com/hashicorp/go-hclog"
	"github.com/incidrthreat/goshorten/backend/config"
	"github.com/incidrthreat/goshorten/backend/shortener"
	"google.golang.org/grpc"
)

func main() {
	log := hclog.Default()
	conf, err := config.ConfigFromFile("config.json")
	if err != nil {
		log.Error("Problem with Json file", "error", err)
		os.Exit(1)
	}

	store := data.Redis{
		CharFloor: conf.Redis.CharFloor,
		Conn: &redis.Options{
			Addr:     conf.Redis.Host,
			Password: conf.Redis.Pass,
			DB:       conf.Redis.DB,
		},
	}

	store.Init()

	lis, err := net.Listen("tcp", conf.GRPCHost)
	if err != nil {
		log.Error("Unable to create listener", "error", err)
		os.Exit(1)
	}

	gs := grpc.NewServer()
	// reflection.Register(gs) // Remove before production

	pb.RegisterShortenerServer(gs, &shortener.CreateServer{
		Store: store,
	})

	log.Info("Serving gRPC", "Host", hclog.Fmt("%s", conf.GRPCHost))
	err = gs.Serve(lis)
	if err != nil {
		log.Error("Serve Error", "Error", err)
	}
}
