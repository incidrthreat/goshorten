<!-- refreshed: 2026-08-31 -->
# Architecture

**Analysis Date:** 2026-08-31

## System Overview

GoShorten is two independently deployed Go binaries ("backend" and "frontend") plus Postgres and Redis, generated from a single gRPC proto contract. The backend is the sole owner of state; the frontend is a thin edge service that proxies REST/API calls and resolves short-code redirects via gRPC.

```text
┌───────────────────────────────────────────────────────────────────────────┐
│  Browser / API client                                                     │
└───────────────┬───────────────────────────────────┬───────────────────────┘
                 │ HTTP :8081                        │ (direct REST, e.g. CI/scripts)
                 ▼                                   │
┌───────────────────────────────────────────────────────────────────────────┐
│  frontend binary  `frontend/cmd/main.go` → `frontend/webapp/`             │
│  ┌─────────────────┐ ┌────────────────────┐ ┌─────────────────────────┐  │
│  │ /api/* proxy     │ │ short-code redirect│ │ React SPA (static)      │  │
│  │ `webapp/proxy.go`│ │ `webapp/grpc.go`   │ │ `webapp/routes.go`      │  │
│  └────────┬─────────┘ └──────────┬─────────┘ └──────────────────────────┘ │
└───────────┼──────────────────────┼────────────────────────────────────────┘
            │ HTTP :8080           │ gRPC :9000 (GetURL only)
            ▼                      ▼
┌───────────────────────────────────────────────────────────────────────────┐
│  backend binary  `backend/main.go`                                        │
│  ┌────────────────────────────┐   ┌───────────────────────────────────┐  │
│  │ REST Gateway (grpc-gateway) │   │ gRPC server :9000                 │  │
│  │ `backend/gateway/gateway.go`│──▶│ AuthInterceptor `auth/interceptor` │  │
│  │ + Admin/self-service REST   │   │ Shortener + Auth services         │  │
│  │ `gateway/admin.go`          │   │ `backend/shortener/`              │  │
│  │ `gateway/members.go`        │   └──────────────┬────────────────────┘  │
│  │ `gateway/workspace.go`      │                  │                       │
│  └──────────────────────────────────────────────────────────────────────┘│
└───────────────────────────────────────────────────┬───────────────────────┘
                                                      ▼
┌───────────────────────────────────────────────────────────────────────────┐
│  Data layer  `backend/data/`, `backend/auth/`                             │
│  CachedStore (write-through) → PostgresStore (source of truth, RLS)       │
│  + Redis (redirect cache)     + AuthStore (users/sessions/workspaces)     │
└───────────────────────────────────────────────────────────────────────────┘
```

## Component Responsibilities

| Component | Responsibility | File |
|-----------|----------------|------|
| Backend entrypoint | Wires stores, auth, gRPC server, REST gateway, signal handling | `backend/main.go` |
| gRPC services | Implements `Shortener` and `Auth` RPCs (business logic) | `backend/shortener/shortener.go`, `backend/shortener/auth_service.go` |
| Auth interceptor | JWT/API-key auth, workspace resolution, viewer read-only enforcement | `backend/auth/interceptor.go` |
| Auth/tenant store | Users, sessions, workspaces, memberships, invitations, OIDC, API keys | `backend/auth/store.go`, `backend/auth/workspace.go`, `backend/auth/membership.go`, `backend/auth/invitation.go`, `backend/auth/oidc.go` |
| REST gateway | grpc-gateway HTTP↔gRPC translation + hand-written admin/self-service REST routes | `backend/gateway/gateway.go`, `backend/gateway/admin.go`, `backend/gateway/members.go`, `backend/gateway/workspace.go` |
| Cached data store | Write-through Redis cache in front of Postgres for redirect lookups | `backend/data/cached_store.go` |
| Postgres store | Source-of-truth persistence, RLS-scoped transactions, migrations runner | `backend/data/postgres.go`, `backend/data/migrate.go` |
| Analytics/tags/visits | Click analytics aggregation, tag management, async visit logging | `backend/data/analytics.go`, `backend/data/tags.go`, `backend/data/visit.go` |
| Mail | SMTP invitation email delivery (log-only fallback) | `backend/mail/mailer.go` |
| Frontend entrypoint | HTTP server wiring, gRPC client to backend | `frontend/cmd/main.go` |
| Frontend router | Priority routing: API proxy → short-code redirect → SPA catch-all | `frontend/webapp/routes.go` |
| Frontend redirect handler | Calls backend `GetURL` RPC directly, issues HTTP redirect | `frontend/webapp/grpc.go` |
| React SPA | Dashboard UI for URL/tag/member/settings management | `frontend/ui-src/src/App.tsx` |
| Proto contract | Single source of truth for RPCs, HTTP bindings, generated into both binaries | `protos/url_service.proto` |

## Pattern Overview

**Overall:** Two-service, gRPC-first backend-for-frontend (BFF) split. The backend exposes gRPC natively and derives a REST API from the same proto via `grpc-gateway`; the frontend is a presentation-tier proxy with no business logic and no direct database access.

**Key Characteristics:**
- Single proto (`protos/url_service.proto`) is the API contract; Go server/client stubs are generated independently into `backend/pb/` and `frontend/pb/` (duplicated, not shared via Go module).
- Backend owns all persistence; frontend never talks to Postgres/Redis.
- Multi-tenancy is enforced at two layers: application-level workspace scoping (interceptor + query params) and Postgres Row-Level Security as a backstop (migration `000008_rls.up.sql`).
- Redis is a cache, not a system of record — Postgres is authoritative and Redis entries are backfilled/invalidated around it (`backend/data/cached_store.go`).
- Auth supports three credential types on the same gRPC interceptor: JWT bearer tokens, opaque API keys, and OIDC (delegated to JWT after callback).

## Layers

**Transport / Entry (backend):**
- Purpose: Accept gRPC and REST traffic, translate REST↔gRPC, terminate connections
- Location: `backend/main.go`, `backend/gateway/`
- Contains: gRPC server setup, grpc-gateway mux, hand-rolled REST handlers for endpoints not modeled in the proto (admin, members, workspace, self-service auth)
- Depends on: `backend/shortener` (gRPC service impls), `backend/auth` (interceptor, stores)
- Used by: External clients (browser SPA, API consumers), `frontend` binary

**Service / RPC implementation:**
- Purpose: Business logic for each RPC — validation, orchestration, response shaping
- Location: `backend/shortener/` (`shortener.go` URL CRUD, `auth_service.go` login/OIDC/sessions/API-keys, `analytics.go`, `tags.go`, `qr.go`, `validate.go`)
- Contains: `CreateServer` (implements `Shortener` service) and `AuthServer` (implements `Auth` service)
- Depends on: `backend/data` (URLStore interface), `backend/auth` (AuthStore, workspace context helpers)
- Used by: gRPC server registration in `backend/main.go`

**Auth / Tenancy:**
- Purpose: Authentication (password, OIDC, API key), session management, workspace/membership/invitation lifecycle, RBAC
- Location: `backend/auth/`
- Contains: `AuthInterceptor` (per-RPC auth), `AuthStore` (Postgres-backed user/session/workspace/membership CRUD), `JWTManager`, `OIDCManager`
- Depends on: `pgxpool.Pool` directly (not the URLStore abstraction)
- Used by: `backend/shortener`, `backend/gateway`, `backend/main.go`

**Data / Persistence:**
- Purpose: Storage abstraction and implementations
- Location: `backend/data/`
- Contains: `URLStore` interface (`store.go`), `PostgresStore` (source of truth, RLS transaction helpers), `CachedStore` (Redis write-through wrapper), `AnalyticsStore`, `TagStore`, `VisitLogger` (buffered async click writer), migration runner
- Depends on: `pgx/v5`, `go-redis/redis`, `golang-migrate`
- Used by: `backend/shortener`, `backend/gateway/admin.go`

**Frontend edge:**
- Purpose: Serve the SPA, proxy `/api/*` to the backend REST gateway, resolve short codes to redirects via direct gRPC call, health check
- Location: `frontend/webapp/` (`app.go` struct, `routes.go` router, `grpc.go` redirect handler, `proxy.go` reverse proxy)
- Depends on: `frontend/pb` (generated gRPC client stubs, separate copy from backend's)
- Used by: `frontend/cmd/main.go`

**Presentation (React SPA):**
- Purpose: Browser dashboard for authenticated URL/tag/member/settings management
- Location: `frontend/ui-src/src/`
- Contains: `pages/` (route-level screens), `components/` (Layout, TagInput, WorkspaceSwitcher), `hooks/useAuth.ts`, `api/client.ts` (fetch wrapper), `types/index.ts`
- Depends on: Backend REST API (`/api/v1/*`) via `fetch`, `react-router-dom` for client-side routing
- Used by: Served as static build output through `frontend/webapp` (mounted at `frontend/webapp` SPA dir, default `./dist`)

## Data Flow

### Short-code redirect (hot path)

1. Browser requests `GET /{code}` → frontend binary router (`frontend/webapp/routes.go:33-70`)
2. Path matches `codePattern` and is not a reserved SPA route → `app.GetURL` (`frontend/webapp/grpc.go:14`)
3. Frontend calls backend gRPC `Shortener/GetURL` directly (bypasses REST gateway) using `frontend/pb` client stubs
4. Backend `CreateServer.GetURL` — public method (no auth required, see `auth/interceptor.go:45`) — resolves via `CachedStore.Load` (`backend/data/cached_store.go`)
5. Redis cache hit → validates against Postgres synchronously (enforces max-visits/active/TTL) → returns cached long URL; miss → Postgres lookup, backfills Redis with 24h TTL
6. Click recorded asynchronously via buffered `VisitLogger` channel (`backend/data/visit.go`), decoupling redirect latency from analytics writes
7. Frontend issues `http.Redirect` with the stored redirect type (301/302/307/308), sets `X-Robots-Tag: noindex` if not crawlable

### Authenticated dashboard request (e.g. list URLs)

1. React SPA calls `fetch('/api/v1/short-urls')` with `Authorization: Bearer <jwt>` and `X-Workspace-Id` headers (`frontend/ui-src/src/api/client.ts:9-24`)
2. Frontend binary's `/api/` prefix routes to `APIProxy` reverse proxy → backend REST gateway `:8080` (`frontend/webapp/proxy.go`, `frontend/webapp/routes.go:38`)
3. Backend `grpc-gateway` mux translates HTTP → gRPC, forwarding `authorization`/`x-workspace-id` headers into gRPC metadata via `customHeaderMatcher` (`backend/gateway/gateway.go:101-108`)
4. `AuthInterceptor.Unary()` runs before the RPC handler: verifies JWT, validates session (JTI), resolves active workspace (header override → session → user default), resolves per-workspace role, rejects viewer writes (`backend/auth/interceptor.go:60-135`)
5. `CreateServer.ListURLs` reads workspace ID from context (`auth.WorkspaceIDFromContext`) and delegates to `CachedStore` → `PostgresStore`, which runs the query inside a `withWS` transaction that sets the `app.current_workspace_id` Postgres session GUC (`backend/data/postgres.go:60-68`) so Row-Level Security scopes the result set

### Admin / self-service REST (outside the proto)

1. Endpoints like invitations, members, workspace CRUD, admin user management are **not** in `protos/url_service.proto` — they are hand-written `net/http` handlers registered directly on the gateway's `http.ServeMux`
2. Entry: `backend/gateway/admin.go` (`AdminHandler.Register`), `backend/gateway/members.go`, `backend/gateway/workspace.go`
3. These call `backend/auth.AuthStore` methods directly (bypassing gRPC and the interceptor); each handler manually calls `requireAuth`/`requireAdmin`/`requireWorkspaceMember` for authz (`backend/gateway/admin.go:44-60`, `backend/gateway/members.go:26-51`)

**State Management (frontend SPA):**
- No global state library. `useAuth` hook (`frontend/ui-src/src/hooks/useAuth.ts`) holds user/session state in component state, persists JWT and active workspace ID to `localStorage` (`token`, `workspaceId` keys)
- Per-page state fetched ad hoc via `frontend/ui-src/src/api/client.ts` on mount

## Key Abstractions

**URLStore interface:**
- Purpose: Decouples RPC handlers from storage implementation
- Examples: `backend/data/store.go` (interface definition), `backend/data/postgres.go` (Postgres impl), `backend/data/cached_store.go` (Redis-caching decorator)
- Pattern: Decorator — `CachedStore.Primary` wraps `*PostgresStore`; both implement `URLStore`, so `CreateServer.Store` is typed as the interface and swappable

**RLS transaction helpers (`withWS` / `withBypass`):**
- Purpose: Every `urls` table access must run inside a Postgres transaction that either pins `app.current_workspace_id` (tenant-scoped) or explicitly sets `app.bypass_rls` (operator/global queries)
- Examples: `backend/data/postgres.go:60-90`
- Pattern: All `PostgresStore` methods accept a `workspaceID` and internally call `withWS`/`withBypass` before executing queries — RLS policies (migration `000008_rls.up.sql`) are the enforcement backstop if application code forgets a workspace filter

**Context-carried auth/tenant identity:**
- Purpose: Pass authenticated user ID, role, session ID, resolved workspace ID, and workspace role through `context.Context` from the interceptor to RPC handlers
- Examples: `backend/auth/interceptor.go` (`UserIDFromContext`, `WorkspaceIDFromContext`, `WorkspaceRoleFromContext`, etc.)
- Pattern: Typed context keys (`contextKey` string type) + accessor functions; handlers call `requireWorkspace(ctx)` (`backend/shortener/shortener.go:36`) rather than re-parsing tokens

**Async visit logging:**
- Purpose: Decouple redirect latency from click-analytics writes
- Examples: `backend/data/visit.go` (`VisitLogger`, buffered channel + worker goroutines), constructed in `backend/main.go:96` with buffer 4096 and 2 workers
- Pattern: Producer (redirect handler) enqueues; background workers batch-insert to Postgres; `Close()` flushes on shutdown

**Generated gRPC/REST stubs (duplicated, not shared):**
- Purpose: Type-safe RPC contracts
- Examples: `backend/pb/url_service.pb.go`, `backend/pb/url_service.pb.gw.go`, `backend/pb/url_service_grpc.pb.go` vs. `frontend/pb/url_service.pb.go`, `frontend/pb/url_service_grpc.pb.go`
- Pattern: Both `backend/pb` and `frontend/pb` are generated from the same `protos/url_service.proto` but are separate Go packages under separate `go.mod` modules — no shared internal Go module; regeneration must be run for both when the proto changes

## Entry Points

**Backend gRPC + REST server:**
- Location: `backend/main.go`
- Triggers: Process start (container `CMD ["./grpc-server"]` in `backend/Containerfile`)
- Responsibilities: Load config, run Postgres migrations, construct stores (Postgres, Redis, cached, analytics, tags), bootstrap break-glass admin, load OIDC providers, resolve default workspace, start gRPC server (`:9000` default) and REST gateway (`:8080` default) concurrently, run hourly invitation-expiry sweep, handle SIGINT/SIGTERM graceful shutdown

**Frontend HTTP server:**
- Location: `frontend/cmd/main.go`
- Triggers: Process start (container `CMD` in `frontend/Containerfile`)
- Responsibilities: Establish gRPC client connection to backend, construct `webapp.App`, serve HTTP on `:8081` default (API proxy, redirect handler, static SPA), graceful shutdown on signal

**Backend healthcheck binary:**
- Location: `backend/healthcheck/main.go`
- Triggers: Docker/k8s `livenessProbe`/`healthcheck` exec probe (`CMD ["./healthcheck"]`)
- Responsibilities: Lightweight standalone binary compiled separately, used by container orchestration instead of curl/wget in the distroless image

**Frontend healthcheck binary:**
- Location: `frontend/healthcheck/main.go`
- Triggers: Same pattern as backend healthcheck, for the frontend container

## Architectural Constraints

- **Threading:** Standard Go concurrency — one goroutine per gRPC/HTTP request (net/http and grpc-go defaults). Background goroutines: REST gateway (`go func()` in `backend/main.go:186`), invitation expiry sweep ticker (`backend/main.go:211`), `VisitLogger` worker pool (2 workers, `backend/data/visit.go`). No custom worker-pool abstraction beyond the visit logger.
- **Global state:** Package-level `var log = hclog.Default()` singletons in multiple packages (`backend/data`, `backend/auth`, `backend/shortener`, `backend/gateway`) — not injected, shared mutable logger reference. `AuthInterceptor.PublicMethods`/`AdminMethods` maps are constructed once at startup and read concurrently without locking (safe because never mutated after `NewAuthInterceptor`).
- **Dual proto stub trees:** `backend/pb` and `frontend/pb` are independently generated from `protos/url_service.proto` — changing the proto requires regenerating both; there is no shared Go module enforcing they stay in sync at compile time.
- **RLS is mandatory, not optional:** Every Postgres query against tenant tables must go through `withWS`/`withBypass` (`backend/data/postgres.go`) — a query issued directly against the pool outside these helpers will fail or silently see zero rows under `FORCE ROW LEVEL SECURITY`.
- **Frontend has no DB/Redis credentials:** It only holds a gRPC client connection and an HTTP reverse-proxy target — this is enforced structurally (webapp package imports nothing from `backend/data`), not just by convention.

## Anti-Patterns

### Hand-written REST routes bypass the gRPC interceptor's authz pipeline

**What happens:** `backend/gateway/admin.go`, `members.go`, `workspace.go` register `net/http` handlers directly on the gateway mux for endpoints that aren't in `protos/url_service.proto` (admin users, invitations, workspace/member CRUD). Each handler re-implements `requireAuth`/`requireAdmin`/`requireWorkspaceMember` inline instead of reusing `auth.AuthInterceptor`.
**Why it's wrong:** Two separate authorization code paths (gRPC interceptor vs. per-handler checks in `gateway/*.go`) must be kept consistent by hand; a new REST-only endpoint that forgets to call `requireAuth` has no interceptor safety net.
**Do this instead:** When adding new admin/self-service endpoints, follow the existing `requireAuth`/`requireAdmin`/`requireWorkspaceMember` helper pattern already established in `backend/gateway/admin.go:44-60` and `backend/gateway/members.go:26-51` rather than inventing new authz logic — do not skip these calls.

### Duplicated generated pb code between backend and frontend

**What happens:** `backend/pb/` and `frontend/pb/` both contain generated Go stubs for the same `protos/url_service.proto`, checked in separately under two different Go modules.
**Why it's wrong:** Proto changes require manually regenerating both trees; drift between them would not be caught by the Go compiler since they're separate modules with separate `go.mod` files.
**Do this instead:** After editing `protos/url_service.proto`, regenerate both `backend/pb` and `frontend/pb` in the same change and verify both `go build` succeed (see CI jobs in `.github/workflows/`).

## Error Handling

**Strategy:** gRPC handlers return `status.Error(codes.X, "message")` from `google.golang.org/grpc/status`; REST-only handlers write JSON error bodies via a `writeJSON(w, statusCode, map[string]string{"error": ...})` helper in `backend/gateway/admin.go`.

**Patterns:**
- Validation errors surfaced as `codes.InvalidArgument` (e.g. `backend/shortener/shortener.go` slug validation)
- Auth failures as `codes.Unauthenticated` / `codes.PermissionDenied`
- Tenant-isolation lookups return `codes.NotFound` even when the record exists in another workspace, to avoid leaking existence across tenants (`getScopedURL`, `backend/shortener/shortener.go:44-56`)
- Non-fatal background failures (cache write failure, OIDC provider registration failure) are logged via `hclog` and swallowed rather than propagated (`backend/data/cached_store.go`, `backend/main.go:132-138`)

## Cross-Cutting Concerns

**Logging:** `hashicorp/go-hclog`, one logger per package (package-level `var log = hclog.Default()`), configurable level/JSON format via `GOSHORTEN_LOG_LEVEL`/`GOSHORTEN_LOG_JSON` env vars (`backend/main.go:41-53`, mirrored in `frontend/cmd/main.go`).

**Validation:** Request-level validation lives in RPC handlers themselves (`backend/shortener/validate.go` for URL/slug rules) rather than a centralized middleware/validator layer.

**Authentication:** Centralized in `backend/auth.AuthInterceptor` for all proto-defined gRPC methods; REST-only endpoints re-implement equivalent checks per-handler (see Anti-Patterns above). Three credential types share the same interceptor entry point: JWT bearer, `ApiKey <key>`, and OIDC (which issues a JWT after callback, then behaves like normal JWT auth).

---

*Architecture analysis: 2026-08-31*
