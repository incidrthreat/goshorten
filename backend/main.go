package main

import (
	"net"
	"net/http"
	"os"

	"github.com/go-redis/redis"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/incidrthreat/goshorten/backend/data"
	pb "github.com/incidrthreat/goshorten/backend/pb"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

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

	log.Info("Server listening on", "Host:Port", hclog.Fmt("%v", conf.GRPCHost))

	gs := grpc.NewServer()
	// reflection.Register(gs) // Remove before production

	pb.RegisterShortenerServer(gs, &shortener.CreateServer{
		Store: store,
	})

	go func() {
		log.Info("Serving gRPC: ", gs.Serve(lis).Error())
	}()

	grpcWebServer := grpcweb.WrapServer(gs)

	httpServer := &http.Server{
		Addr: conf.GRPCProxyAddr,
		Handler: h2c.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ProtoMajor == 2 {
				grpcWebServer.ServeHTTP(w, r)
			} else {
				w.Header().Set("Access-Control-Allow-Origin", "*")
				w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
				w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-User-Agent, X-Grpc-Web")
				w.Header().Set("grpc-status", "")
				w.Header().Set("grpc-message", "")
				if grpcWebServer.IsGrpcWebRequest(r) {
					grpcWebServer.ServeHTTP(w, r)
				}
			}
		}), &http2.Server{}),
	}
	log.Info("Serving Proxy: ", httpServer.ListenAndServe().Error())

}
