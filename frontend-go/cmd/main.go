package main

import (
	"context"
	"net/http"
	"time"

	"google.golang.org/grpc/keepalive"

	"github.com/hashicorp/go-hclog"
	"github.com/incidrthreat/goshorten/frontend-go/webapp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	port      string = ":8081"
	htmlDir   string = "./ui/templates"
	staticDir string = "./ui/static"
	version   string = "1.0.4"
)

func main() {
	log := hclog.Default()

	clientCert, err := credentials.NewClientTLSFromFile("server.crt", "")
	if err != nil {
		log.Error("Failed to create Certificate", "Error", err)
	}

	var kaCP = keepalive.ClientParameters{
		Time:                15 * time.Second,
		Timeout:             5 * time.Second,
		PermitWithoutStream: true,
	}

	conn, err := grpc.DialContext(context.Background(), "grpcbackend:9000", grpc.WithTransportCredentials(clientCert), grpc.WithKeepaliveParams(kaCP))
	// TODO: Better error handling and keep-alive
	if err != nil {
		log.Error("Failed to connect to gRPC Server", "Error", err)
	}
	defer conn.Close()

	app := &webapp.App{
		HTMLDir:   htmlDir,
		StaticDir: staticDir,
		Conn:      conn,
	}

	log.Info("Starting URL Shortener Client", "Version/Port", hclog.Fmt("%s/%s", version, port))

	err = http.ListenAndServe(port, app.Routes())
	log.Error("Failed to listen and serve HTTP", "Error", err)

}
