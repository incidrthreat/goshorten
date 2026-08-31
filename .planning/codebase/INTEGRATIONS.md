# External Integrations

**Analysis Date:** 2026-08-31

## APIs & External Services

**Identity/SSO:**
- OpenID Connect (OIDC) — any OIDC-compliant identity provider (Google, Okta, Auth0, Keycloak, etc.), configured dynamically per-workspace/instance rather than hardcoded
  - SDK/Client: `github.com/coreos/go-oidc/v3` + `golang.org/x/oauth2`
  - Implementation: `backend/auth/oidc.go` (`OIDCManager`, `RegisterProvider`, provider discovery via `oidc.NewProvider`)
  - Admin-managed providers stored in Postgres and loaded/registered at server boot (`backend/main.go` — `authStore.ListOIDCProviders`, `oidcMgr.RegisterProvider`); can also be registered live via admin API without restart (`backend/gateway/admin.go:791-793`)
  - REST admin endpoints: `GET/POST /api/v1/admin/oidc-providers`, `/api/v1/admin/oidc-providers/{name}` (`backend/gateway/admin.go:78-79`)
  - Auth: per-provider `ClientID`/`ClientSecret`/`IssuerURL`/`RedirectURI`/`Scopes` stored in the `oidc_providers` Postgres table (not env vars) — see `backend/auth/oidc.go` struct `OIDCProviderConfig`
  - User records track OIDC origin via `OIDCIssuer` field, surfaced in `GET /api/v1/auth/account` (`backend/gateway/admin.go:500-530`)

**API Documentation:**
- Swagger UI — third-party CDN assets loaded client-side (`unpkg.com/swagger-ui-dist@5`), served at `/api/v1/docs` (`backend/gateway/swagger_ui.go`), backed by generated spec `backend/pb/url_service.swagger.json` served at `/api/v1/swagger.json` (`backend/gateway/gateway.go`)

## Data Storage

**Databases:**
- PostgreSQL 16 (`postgres:16-alpine`) — system of record for URLs, users, workspaces, memberships, analytics, tags, OIDC provider configs
  - Client: `github.com/jackc/pgx/v5` (`pgxpool.Pool`), `backend/data/postgres.go`
  - Connection: `PostgresConf.DSN()` built from `GOSHORTEN_POSTGRES_{HOST,PORT,USER,PASSWORD,DB}` env vars or `config.json` (`backend/config/config.go`)
  - Migrations: `github.com/golang-migrate/migrate/v4`, 9 versioned SQL migration pairs in `backend/migrations/` (schema → URL features → auth → analytics → phase12 → system settings → multitenancy → RLS → memberships); run automatically at server startup (`backend/data/migrate.go`, called from `backend/main.go`)
  - Row-Level Security (RLS) used for tenant isolation — see `backend/migrations/000008_rls.up.sql` and `pgxQuerier` abstraction supporting RLS-scoped transactions (`backend/data/postgres.go`)

**Caching:**
- Redis (`redis:alpine`) — cache layer only, NOT source of truth (explicit design per `backend/main.go` comment "Initialize Redis client (cache only)")
  - Client: `github.com/go-redis/redis` v6, `backend/data/redis.go`
  - Composed via `CachedStore` wrapping the Postgres store with Redis caching (`backend/data/cached_store.go`)
  - Connection: `GOSHORTEN_REDIS_HOST`, `GOSHORTEN_REDIS_PASS` env vars or `config.json` `redis_conf`
  - Password-protected in both docker-compose (`redis-server --requirepass`) and k8s (`REDIS_PASSWORD` from Secret)

**File Storage:**
- None — no object storage (S3/GCS/etc.) integration detected; QR codes are generated on-demand in-memory (`backend/shortener/qr.go`) and streamed to the response, not persisted to disk or blob storage

## Authentication & Identity

**Auth Provider:**
- Custom (self-hosted, no external auth-as-a-service like Auth0/Clerk)
  - Password auth: bcrypt (cost 12) via `golang.org/x/crypto/bcrypt`, `backend/auth/password.go`
  - Session tokens: JWT (`golang-jwt/jwt/v5`), `backend/auth/jwt.go`, secret from `GOSHORTEN_JWT_SECRET` env var or `config.json` `auth_conf.jwt_secret`
  - API keys: server-generated 32-byte random hex, stored as SHA-256 hash (`backend/auth/password.go` — `GenerateAPIKey`, `HashAPIKey`)
  - SSO: OIDC (see APIs & External Services above), can fully replace password login via `GOSHORTEN_DISABLE_PASSWORD_LOGIN=true`
  - Break-glass admin account bootstrapped from `GOSHORTEN_ADMIN_EMAIL`/`GOSHORTEN_ADMIN_PASSWORD` at first boot (`backend/main.go` — `authStore.BootstrapAdmin`)
  - Multi-tenancy: workspaces + memberships + invitations (`backend/auth/workspace.go`, `backend/auth/membership.go`, `backend/auth/invitation.go`)

## Monitoring & Observability

**Error Tracking:**
- None — no Sentry/Rollbar/etc. integration detected

**Logs:**
- `github.com/hashicorp/go-hclog` structured logging in both backend (`backend/main.go`) and frontend (`frontend/cmd/main.go`) Go services
- Level and format configurable via `GOSHORTEN_LOG_LEVEL` and `GOSHORTEN_LOG_JSON` (JSON logs for log-aggregator ingestion)
- No external log-shipping integration configured in code (left to platform/k8s log collection)

**Health Checks:**
- Custom `healthcheck` binaries (`backend/healthcheck/main.go`, `frontend/healthcheck/main.go`) compiled separately and used as container `HEALTHCHECK`/k8s probe commands
- REST endpoints `/healthz` and `/readyz` (gateway `ReadyCheckers` ping Postgres and Redis — `backend/main.go`, `backend/gateway/gateway.go`)

## CI/CD & Deployment

**Hosting:**
- Self-hosted/BYO Kubernetes — manifests in `k8s/` (`namespace.yaml`, `backend.yaml`, `frontend.yaml`, `postgres.yaml`, `redis.yaml`, `secret.yaml`) with an nginx-class `Ingress`
- Container registry: GitHub Container Registry (GHCR) — `ghcr.io/incidrthreat/goshorten-backend`, `ghcr.io/incidrthreat/goshorten-frontend`

**CI Pipeline:**
- GitHub Actions (`.github/workflows/ci.yml`) — three parallel jobs on every push/PR: backend (golangci-lint, build+test+docker-build gated on release events), frontend (same pattern), UI (npm install, `tsc --noEmit`, build gated on release)
- GitHub Actions (`.github/workflows/publish.yml`) — triggered on `v*` tags; builds and pushes multi-arch (amd64/arm64) Docker images to GHCR using `docker/build-push-action@v6` + `docker buildx imagetools create` for multi-arch manifests; auth via `secrets.GITHUB_TOKEN`

## Environment Configuration

**Required env vars (backend, prefix `GOSHORTEN_`):**
- `REDIS_HOST`, `REDIS_PASS`
- `POSTGRES_HOST`, `POSTGRES_PORT`, `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB`
- `GRPC_HOST`, `GATEWAY_ADDR`
- `JWT_SECRET`, `ADMIN_EMAIL`, `ADMIN_PASSWORD`, `DISABLE_PASSWORD_LOGIN`
- `APP_BASE_URL` (public origin for invite links)
- `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM` (optional — see Webhooks & Callbacks below)
- `LOG_LEVEL`, `LOG_JSON`

**Required env vars (frontend Go service):**
- `GOSHORTEN_FRONTEND_PORT`, `GOSHORTEN_GRPC_ADDR`, `GOSHORTEN_BACKEND_URL`, `GOSHORTEN_SPA_DIR`, `GOSHORTEN_LOG_LEVEL`, `GOSHORTEN_LOG_JSON`

**Secrets location:**
- Local dev: `docker-compose.yml` inline plaintext env vars (dev-only, weak defaults like `mysecretpassword`/`admin`)
- Production: Kubernetes `Secret` object `goshorten-secrets` (`k8s/secret.yaml`), referenced via `secretKeyRef` in `k8s/backend.yaml`, `k8s/postgres.yaml`, `k8s/redis.yaml`; repo comment recommends Sealed Secrets/External Secrets/SOPS for real deployments, values in the committed file are placeholders (`changeme`)
- Backend also supports a `config.json` file (`backend/config.json.example`) as the base config, with env vars always taking precedence (`backend/config/config.go` — `applyEnvOverrides`)

## Webhooks & Callbacks

**Incoming:**
- OIDC callback endpoints for SSO login completion (exact route defined per-provider `RedirectURI`, handled server-side; see `backend/auth/oidc.go` and OIDC-related handlers in `backend/shortener/auth_service.go`)
- No third-party webhook receivers (e.g. Stripe, GitHub webhooks) detected

**Outgoing:**
- SMTP email — outbound-only integration for workspace invitations (`backend/mail/mailer.go`)
  - Uses Go stdlib `net/smtp`; supports STARTTLS (port 587) and implicit TLS (port 465)
  - Gracefully degrades: if `GOSHORTEN_SMTP_HOST` (or `mail_conf.host`) is empty, `Mailer.Enabled()` returns false and the mailer logs the message (including the invite accept link) instead of sending — invitations still function without SMTP configured, and the UI also shows a copyable link
  - Entry point: `mail.New(mail.Config{...})` constructed in `backend/main.go` from `conf.Mail`, used by `AdminHandler.Mailer` in `backend/gateway/admin.go` for `SendInvitation`
- OIDC discovery/token requests — outbound HTTPS calls to configured identity providers' `.well-known/openid-configuration`, authorization, and token endpoints (`backend/auth/oidc.go`, via `golang.org/x/oauth2` and `coreos/go-oidc`)

---

*Integration audit: 2026-08-31*
