package main

import (
	"net"
	"os"

	"github.com/go-redis/redis"
	"github.com/incidrthreat/goshorten/backend/server/data"
	pb "github.com/incidrthreat/goshorten/protos"

	"github.com/hashicorp/go-hclog"
	"github.com/incidrthreat/goshorten/backend/server/shortener"
	"github.com/incidrthreat/goshorten/config"
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

	log.Info("Server listening on", "Host:Port", hclog.Fmt("%v", conf.GRPCHost))

	gs := grpc.NewServer()
	reflection.Register(gs) // Remove before production

	pb.RegisterShortenerServer(gs, &shortener.CreateServer{
		Store: store,
	})

	gs.Serve(lis)

}
