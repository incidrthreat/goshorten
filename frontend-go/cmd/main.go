package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"google.golang.org/grpc/keepalive"

	"github.com/hashicorp/go-hclog"
	"github.com/incidrthreat/goshorten/frontend-go/webapp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const version = "3.0.0"

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	log := hclog.Default()

	port := envOrDefault("GOSHORTEN_FRONTEND_PORT", ":8081")
	grpcAddr := envOrDefault("GOSHORTEN_GRPC_ADDR", "grpcbackend:9000")
	backendURL := envOrDefault("GOSHORTEN_BACKEND_URL", "http://grpcbackend:8080")
	spaDir := envOrDefault("GOSHORTEN_SPA_DIR", "./dist")

	var kaCP = keepalive.ClientParameters{
		Time:                15 * time.Second,
		Timeout:             5 * time.Second,
		PermitWithoutStream: true,
	}

	conn, err := grpc.DialContext(context.Background(), grpcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(kaCP),
	)
	if err != nil {
		log.Error("Failed to connect to gRPC Server", "Error", err)
	}
	defer conn.Close()

	app := &webapp.App{
		SPADir:     spaDir,
		BackendURL: backendURL,
		Conn:       conn,
	}

	log.Info("GoShorten Frontend",
		"Version", version,
		"Port", port,
		"gRPC", grpcAddr,
		"Backend", backendURL,
		"SPA", spaDir,
	)

	err = http.ListenAndServe(port, app.Routes())
	log.Error("Failed to listen and serve HTTP", "Error", err)
}
