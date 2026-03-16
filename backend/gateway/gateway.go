package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/hashicorp/go-hclog"
	pb "github.com/incidrthreat/goshorten/backend/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var log = hclog.Default()

// Config holds the REST gateway configuration.
type Config struct {
	// HTTPAddr is the address the REST gateway listens on (e.g., ":8080").
	HTTPAddr string
	// GRPCAddr is the backend gRPC server address (e.g., "localhost:9000").
	GRPCAddr string
	// SwaggerJSON is the path to the OpenAPI spec file to serve.
	SwaggerJSON string
	// ReadyCheckers are named functions called by /readyz. A non-nil error
	// from any checker causes the endpoint to return 503.
	ReadyCheckers map[string]func(context.Context) error
}

// Run starts the REST gateway HTTP server.
func Run(ctx context.Context, cfg Config) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := runtime.NewServeMux(
		runtime.WithIncomingHeaderMatcher(customHeaderMatcher),
	)

	// Plaintext connection to the co-located gRPC server
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	// Register both services on the gateway
	if err := pb.RegisterShortenerHandlerFromEndpoint(ctx, mux, cfg.GRPCAddr, opts); err != nil {
		return fmt.Errorf("register shortener gateway: %w", err)
	}
	if err := pb.RegisterAuthHandlerFromEndpoint(ctx, mux, cfg.GRPCAddr, opts); err != nil {
		return fmt.Errorf("register auth gateway: %w", err)
	}

	// Build the HTTP handler chain
	handler := corsMiddleware(mux)

	// Serve OpenAPI spec at /api/v1/swagger.json
	httpMux := http.NewServeMux()
	if cfg.SwaggerJSON != "" {
		httpMux.HandleFunc("/api/v1/swagger.json", func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, cfg.SwaggerJSON)
		})
	}
	// Liveness check — always 200 if the process is up
	httpMux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})
	// Readiness check — 200 only when all dependencies are reachable
	httpMux.HandleFunc("/readyz", readyzHandler(cfg.ReadyCheckers))
	// All other requests go to the grpc-gateway mux
	httpMux.Handle("/", handler)

	server := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: httpMux,
	}

	log.Info("REST Gateway", "Listening", cfg.HTTPAddr)

	go func() {
		<-ctx.Done()
		server.Shutdown(context.Background())
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("rest gateway: %w", err)
	}
	return nil
}

// customHeaderMatcher forwards the Authorization header from HTTP to gRPC metadata.
func customHeaderMatcher(key string) (string, bool) {
	switch strings.ToLower(key) {
	case "authorization":
		return "authorization", true
	default:
		return runtime.DefaultHeaderMatcher(key)
	}
}

// readyzHandler returns an HTTP handler that checks all named ready functions.
// It responds with JSON: {"status":"ok"} on 200 or {"status":"degraded","checks":{...}} on 503.
func readyzHandler(checkers map[string]func(context.Context) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		type result struct {
			Status string `json:"status"`
			Error  string `json:"error,omitempty"`
		}
		checks := make(map[string]result, len(checkers))
		allOK := true
		for name, fn := range checkers {
			if err := fn(ctx); err != nil {
				checks[name] = result{Status: "fail", Error: err.Error()}
				allOK = false
			} else {
				checks[name] = result{Status: "ok"}
			}
		}

		w.Header().Set("Content-Type", "application/json")
		if allOK {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "degraded", "checks": checks})
	}
}

// corsMiddleware adds CORS headers for browser access.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
