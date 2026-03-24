package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-redis/redis"
	"github.com/incidrthreat/goshorten/backend/auth"
	"github.com/incidrthreat/goshorten/backend/data"
	"github.com/incidrthreat/goshorten/backend/gateway"
	pb "github.com/incidrthreat/goshorten/backend/pb"

	"github.com/hashicorp/go-hclog"
	"github.com/incidrthreat/goshorten/backend/config"
	"github.com/incidrthreat/goshorten/backend/shortener"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

var version = "dev"

var kaEP = keepalive.EnforcementPolicy{
	MinTime:             5 * time.Second,
	PermitWithoutStream: true,
}

var kaSP = keepalive.ServerParameters{
	MaxConnectionIdle:     15 * time.Second,
	MaxConnectionAge:      30 * time.Second,
	MaxConnectionAgeGrace: 5 * time.Second,
	Time:                  5 * time.Second,
	Timeout:               1 * time.Second,
}

func newLogger() hclog.Logger {
	level := hclog.LevelFromString(os.Getenv("GOSHORTEN_LOG_LEVEL"))
	if level == hclog.NoLevel {
		level = hclog.Info
	}
	jsonFormat := os.Getenv("GOSHORTEN_LOG_JSON") == "true"
	log := hclog.New(&hclog.LoggerOptions{
		Name:       "goshorten",
		Level:      level,
		JSONFormat: jsonFormat,
	})
	hclog.SetDefault(log)
	return log
}

func main() {
	log := newLogger()

	conf, err := config.ConfigFromFile("config.json")
	if err != nil {
		log.Error("Problem with Json file", "error", err)
		os.Exit(1)
	}

	log.Info("GoShorten URL Shortener Server", "Version", version)

	// --- Run Postgres migrations ---
	pgDSN := conf.Postgres.DSN()
	if err := data.RunMigrations(pgDSN, "migrations"); err != nil {
		log.Error("Migration failed", "error", err)
		os.Exit(1)
	}

	// --- Initialize Postgres store (source of truth) ---
	pgStore, err := data.NewPostgresStore(pgDSN, conf.Redis.CharFloor)
	if err != nil {
		log.Error("Postgres init failed", "error", err)
		os.Exit(1)
	}

	// --- Initialize Redis client (cache only) ---
	redisClient := redis.NewClient(&redis.Options{
		Addr:     conf.Redis.Host,
		Password: conf.Redis.Pass,
		DB:       conf.Redis.DB,
	})
	if _, err := redisClient.Ping().Result(); err != nil {
		log.Error("Redis ping failed", "error", err)
		os.Exit(1)
	}
	log.Info("Redis Server", "Connection", "Online (cache mode)")

	// --- Initialize visit logger (async click recording pipeline) ---
	visitLogger := data.NewVisitLogger(pgStore.Pool, 4096, 2)
	pgStore.SetVisitLogger(visitLogger)

	// --- Initialize analytics store ---
	analyticsStore := &data.AnalyticsStore{Pool: pgStore.Pool}

	// --- Initialize tag store ---
	tagStore := &data.TagStore{Pool: pgStore.Pool}

	// --- Compose the cached store ---
	store := &data.CachedStore{
		Primary: pgStore,
		Cache:   redisClient,
	}

	// --- Initialize auth ---
	authStore := auth.NewAuthStore(pgStore.Pool)
	jwtMgr := auth.NewJWTManager(conf.Auth.GetJWTSecret(), conf.Auth.GetTokenExpiry())
	oidcMgr := auth.NewOIDCManager()

	// Bootstrap break-glass admin
	created, err := authStore.BootstrapAdmin(context.Background(),
		conf.Auth.GetAdminEmail(), conf.Auth.GetAdminPassword())
	if err != nil {
		log.Error("Admin bootstrap failed", "error", err)
		os.Exit(1)
	}
	if created {
		log.Info("Auth", "Break-glass admin created", conf.Auth.GetAdminEmail())
	}

	// Load OIDC providers from database
	oidcConfigs, err := authStore.ListOIDCProviders(context.Background())
	if err != nil {
		log.Warn("OIDC", "Failed to load providers", err)
	} else {
		for _, cfg := range oidcConfigs {
			if err := oidcMgr.RegisterProvider(context.Background(), cfg); err != nil {
				log.Warn("OIDC", "Failed to register provider", cfg.Name, "error", err)
			} else {
				log.Info("OIDC", "Provider registered", cfg.Name)
			}
		}
	}

	// --- Auth interceptor ---
	interceptor := auth.NewAuthInterceptor(jwtMgr, authStore)

	lis, err := net.Listen("tcp", conf.GRPCHost)
	if err != nil {
		log.Error("Unable to create listener", "error", err)
		os.Exit(1)
	}

	// Plaintext gRPC — TLS termination happens at the edge (Phase 8)
	gs := grpc.NewServer(
		grpc.KeepaliveEnforcementPolicy(kaEP),
		grpc.KeepaliveParams(kaSP),
		grpc.UnaryInterceptor(interceptor.Unary()),
	)

	// Register URL service
	pb.RegisterShortenerServer(gs, &shortener.CreateServer{
		Store:     store,
		Analytics: analyticsStore,
		Tags:      tagStore,
	})

	// Register Auth service
	pb.RegisterAuthServer(gs, &shortener.AuthServer{
		AuthStore: authStore,
		JWTMgr:    jwtMgr,
		OIDCMgr:   oidcMgr,
	})

	// --- Cancellable root context (drives gateway shutdown) ---
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// --- Start REST API Gateway in background ---
	if conf.Gateway.HTTPAddr != "" {
		gwCfg := gateway.Config{
			HTTPAddr:    conf.Gateway.HTTPAddr,
			GRPCAddr:    conf.GRPCHost,
			SwaggerJSON: "pb/url_service.swagger.json",
			ReadyCheckers: map[string]func(context.Context) error{
				"postgres": func(ctx context.Context) error { return pgStore.Pool.Ping(ctx) },
				"redis":    func(ctx context.Context) error { return redisClient.Ping().Err() },
			},
			AdminHandler: &gateway.AdminHandler{
				AuthStore:            authStore,
				JWTMgr:               jwtMgr,
				URLStore:             store,
				OIDCMgr:              oidcMgr,
				DisablePasswordLogin: conf.Auth.GetDisablePasswordLogin(),
			},
		}
		go func() {
			if err := gateway.Run(ctx, gwCfg); err != nil {
				log.Error("REST Gateway failed", "error", err)
			}
		}()
	}

	// --- Signal handling ---
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		log.Info("Shutdown signal received", "signal", sig.String())
		gs.GracefulStop() // drain in-flight RPCs; causes gs.Serve to return
	}()

	log.Info("Serving gRPC", "Host", hclog.Fmt("%s", conf.GRPCHost))
	if err := gs.Serve(lis); err != nil {
		log.Error("Serve Error", "error", err)
	}

	// gRPC server stopped — shut down remaining services in order
	log.Info("Draining services")
	cancel()             // stop REST gateway HTTP server
	visitLogger.Close() // flush buffered visit writes
	pgStore.Pool.Close()
	log.Info("Shutdown complete")
}
