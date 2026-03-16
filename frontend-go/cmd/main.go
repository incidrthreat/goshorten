package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc/keepalive"

	"github.com/hashicorp/go-hclog"
	"github.com/incidrthreat/goshorten/frontend-go/webapp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const version = "0.5.0"

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func newLogger() hclog.Logger {
	level := hclog.LevelFromString(os.Getenv("GOSHORTEN_LOG_LEVEL"))
	if level == hclog.NoLevel {
		level = hclog.Info
	}
	jsonFormat := os.Getenv("GOSHORTEN_LOG_JSON") == "true"
	log := hclog.New(&hclog.LoggerOptions{
		Name:       "goshorten-frontend",
		Level:      level,
		JSONFormat: jsonFormat,
	})
	hclog.SetDefault(log)
	return log
}

func main() {
	log := newLogger()

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

	srv := &http.Server{
		Addr:    port,
		Handler: app.Routes(),
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		log.Info("Shutdown signal received", "signal", sig.String())
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Error("HTTP shutdown error", "error", err)
		}
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Error("Failed to listen and serve HTTP", "Error", err)
	}
	log.Info("Frontend shutdown complete")
}
