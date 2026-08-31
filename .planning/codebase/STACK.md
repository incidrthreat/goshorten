# Technology Stack

**Analysis Date:** 2026-08-31

## Languages

**Primary:**
- Go 1.25.0 — backend gRPC/REST server (`backend/go.mod`, module `github.com/incidrthreat/goshorten/backend`)
- Go 1.24.0 (toolchain 1.24.2) — frontend SPA-serving/gRPC-proxy server (`frontend/go.mod`, module `github.com/incidrthreat/goshorten/frontend`)
- TypeScript 5.8 — React single-page app (`frontend/ui-src/tsconfig.json`, `frontend/ui-src/src/`)

**Secondary:**
- SQL — Postgres migrations (`backend/migrations/*.up.sql`, `*.down.sql`), 9 versioned migrations from initial schema through memberships
- Protocol Buffers — gRPC service/message definitions (`protos/url_service.proto`)
- Bash — TLS cert generation helper (`generate-tls-certs.sh`)

There is no single root Go module/workspace — `backend` and `frontend` are two independent Go modules built and versioned separately (each with its own `go.mod`/`go.sum`). No `go.work` file is present.

## Runtime

**Environment:**
- Go 1.25 (backend build image `golang:1.25-alpine`, `backend/Containerfile`)
- Go 1.24 (frontend build image also uses `golang:1.25-alpine` at build time per `frontend/Containerfile`, but `go.mod` pins `go 1.24.0`)
- Node.js 22 (`frontend/Containerfile` build stage `node:22-alpine`; CI uses `actions/setup-node@v4` with `node-version: 22`, `.github/workflows/ci.yml`)
- Both Go services ship as static binaries in `gcr.io/distroless/static-debian12:nonroot` final images (no shell, no package manager, runs as non-root)

**Package Manager:**
- Go modules (`go mod download`) — lockfiles `backend/go.sum`, `frontend/go.sum` present
- npm — lockfile `frontend/ui-src/package-lock.json` (referenced in `frontend/Containerfile` as `package-lock.json*`)

## Frameworks

**Core (backend):**
- `google.golang.org/grpc` v1.79.3 — gRPC server, keepalive-tuned (`backend/main.go`)
- `github.com/grpc-ecosystem/grpc-gateway/v2` v2.28.0 — REST/JSON gateway that proxies to gRPC (`backend/gateway/gateway.go`)
- `github.com/jackc/pgx/v5` v5.5.4 — Postgres driver/connection pool (`backend/data/postgres.go`)
- `github.com/go-redis/redis` v6.15.9+incompatible — Redis client, used as a cache layer only, not source of truth (`backend/data/redis.go`, `backend/data/cached_store.go`)
- `github.com/golang-migrate/migrate/v4` v4.17.0 — Postgres schema migrations run at boot (`backend/data/migrate.go`)
- `github.com/golang-jwt/jwt/v5` v5.3.1 — JWT session tokens (`backend/auth/jwt.go`)
- `github.com/coreos/go-oidc/v3` v3.17.0 + `golang.org/x/oauth2` v0.36.0 — OIDC/SSO provider integration (`backend/auth/oidc.go`)
- `golang.org/x/crypto` v0.46.0 (bcrypt) — password hashing, cost factor 12 (`backend/auth/password.go`)
- `github.com/boombuler/barcode` v1.1.0 — QR code generation for short links (`backend/shortener/qr.go`)
- `github.com/hashicorp/go-hclog` v0.14.1 — structured logging (used by both backend and frontend)

**Core (frontend Go service):**
- `google.golang.org/grpc` v1.79.3 — gRPC client to backend (`frontend/webapp/grpc.go`)
- Standard library `net/http/httputil.ReverseProxy` — proxies `/api/*` to backend REST gateway (`frontend/webapp/proxy.go`)
- Serves the built React SPA as static files plus proxies API/gRPC traffic (`frontend/webapp/routes.go`)

**Frontend UI:**
- React 19.1 + `react-dom` 19.1 — UI framework (`frontend/ui-src/package.json`)
- `react-router-dom` 7.6 — client-side routing
- Vite 6.3 — dev server and bundler (`frontend/ui-src/vite.config.ts`), dev proxy `/api` → `http://localhost:8080`
- Tailwind CSS 3.4 + PostCSS 8.5 + Autoprefixer 10.4 — styling (`frontend/ui-src/tailwind.config.js`, `postcss.config.js`)
- `recharts` 2.15 — analytics charts/dashboards
- `qr-code-styling` 1.6-rc — client-side QR rendering
- `lucide-react` 0.511 — icon set

**Testing:**
- Go standard `testing` package — unit and integration tests (`backend/integration/isolation_test.go`, `backend/integration/membership_test.go`)
- TypeScript type-checking only in CI (`npx tsc --noEmit`) — no JS/TS test runner (Jest/Vitest) configured

**Build/Dev:**
- `protoc` + `protoc-gen-go`, `protoc-gen-go-grpc`, `protoc-gen-grpc-gateway`, `protoc-gen-openapiv2` — proto → Go/OpenAPI codegen (generated output committed at `backend/pb/`, `frontend/pb/`; source protos in `protos/`)
- `golangci-lint` (latest, `goinstall` mode) — Go linting in CI (`.github/workflows/ci.yml`)
- Docker Buildx multi-arch (amd64/arm64) builds — `.github/workflows/publish.yml`
- Podman/Docker `Containerfile` naming convention (both backend and frontend), multi-stage builds

## Key Dependencies

**Critical:**
- `github.com/jackc/pgx/v5` v5.5.4 — sole Postgres access layer; source of truth for URLs, users, workspaces, analytics (`backend/data/postgres.go`)
- `github.com/go-redis/redis` v6 — cache-only layer sitting in front of Postgres (`backend/data/cached_store.go`); NOT the system of record (design changed from earlier Redis-primary architecture)
- `github.com/golang-jwt/jwt/v5` — all session auth (`backend/auth/jwt.go`)
- `github.com/coreos/go-oidc/v3` — enterprise SSO login flows, multi-provider (`backend/auth/oidc.go`)
- `golang.org/x/crypto/bcrypt` — password storage security boundary

**Infrastructure:**
- `github.com/golang-migrate/migrate/v4` — schema versioning, runs automatically at server startup (`backend/data/migrate.go`, invoked in `backend/main.go`)
- `github.com/grpc-ecosystem/grpc-gateway/v2` — single source of truth (proto) drives both gRPC and REST/OpenAPI APIs
- `github.com/hashicorp/go-hclog` — leveled/JSON logging across both services, configurable via `GOSHORTEN_LOG_LEVEL` / `GOSHORTEN_LOG_JSON`

## Configuration

**Environment:**
- Backend reads `config.json` (see `backend/config.json.example`) at startup, then applies environment-variable overrides via `applyEnvOverrides` (`backend/config/config.go`)
- Key env vars (all prefixed `GOSHORTEN_`): `REDIS_HOST`, `REDIS_PASS`, `POSTGRES_HOST`, `POSTGRES_PORT`, `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB`, `GRPC_HOST`, `GATEWAY_ADDR`, `JWT_SECRET`, `ADMIN_EMAIL`, `ADMIN_PASSWORD`, `DISABLE_PASSWORD_LOGIN`, `APP_BASE_URL`, `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM`, `LOG_LEVEL`, `LOG_JSON`
- Frontend Go service env vars: `GOSHORTEN_FRONTEND_PORT` (default `:8081`), `GOSHORTEN_GRPC_ADDR` (default `grpcbackend:9000`), `GOSHORTEN_BACKEND_URL` (default `http://grpcbackend:8080`), `GOSHORTEN_SPA_DIR` (default `./dist`), `GOSHORTEN_LOG_LEVEL`, `GOSHORTEN_LOG_JSON`
- `.env*` files: none detected in repo; `docker-compose.yml` inlines dev-only credentials directly (e.g. `mysecretpassword`, `goshorten_secret`, `admin`) — acceptable only for local dev, k8s uses Secret objects instead
- Frontend UI build-time var: `VITE_APP_VERSION` (injected by Vite build in `frontend/Containerfile` and CI)

**Build:**
- `backend/Containerfile` — multi-stage: `golang:1.25-alpine` builder → `gcr.io/distroless/static-debian12:nonroot` runtime; builds `grpc-server` and a separate `healthcheck` binary; copies `migrations/`, `pb/url_service.swagger.json`, and renames `config.json.example` → `config.json`
- `frontend/Containerfile` — three-stage: `node:22-alpine` (Vite build of React SPA) → `golang:1.25-alpine` (Go server build) → `gcr.io/distroless/static-debian12:nonroot` runtime; copies built SPA `dist/` alongside the Go binary
- `frontend/ui-src/tsconfig.json`, `vite.config.ts`, `tailwind.config.js`, `postcss.config.js` — frontend build config
- `VERSION` file at repo root (currently `0.5.4-rc`) — injected into both Go binaries via `-ldflags="-X main.version=$VER"` and into the frontend build via `VITE_APP_VERSION`

## Platform Requirements

**Development:**
- Go 1.24+/1.25 toolchains for backend and frontend modules respectively
- Node.js 22 + npm for the React UI
- Docker/Podman + `docker-compose.yml` for local full-stack dev (Postgres 16, Redis, backend, frontend containers)
- `protoc` toolchain only needed when regenerating `pb/` code from `protos/*.proto` (generated files are committed, so not required for normal builds)

**Production:**
- Kubernetes (manifests in `k8s/`: `namespace.yaml`, `backend.yaml`, `frontend.yaml`, `postgres.yaml`, `redis.yaml`, `secret.yaml`) with an nginx `Ingress` fronting the frontend service
- Container images published to GHCR: `ghcr.io/incidrthreat/goshorten-backend`, `ghcr.io/incidrthreat/goshorten-frontend`, multi-arch (linux/amd64, linux/arm64) via `.github/workflows/publish.yml`, triggered on `v*` git tags
- Postgres 16 (image `postgres:16-alpine`) with a `PersistentVolumeClaim` (5Gi) in k8s
- Redis (image `redis:alpine`), password-protected, used purely as a cache (no persistent volume in k8s)
- TLS termination happens at the edge/ingress, not in the Go services — gRPC server runs plaintext internally (`backend/main.go` comment: "Plaintext gRPC — TLS termination happens at the edge")

---

*Stack analysis: 2026-08-31*
