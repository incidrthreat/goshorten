# Codebase Structure

**Analysis Date:** 2026-08-31

## Directory Layout

```
goshorten/
├── backend/                  # Go gRPC + REST API server (source of truth for data)
│   ├── auth/                 # JWT/OIDC/API-key auth, sessions, workspaces, memberships, invitations
│   ├── config/                # Configuration struct + env-var overrides
│   ├── data/                 # Storage layer: Postgres store, Redis cache, analytics, tags, visit logging
│   ├── gateway/               # REST gateway: grpc-gateway mux + hand-written admin/self-service routes
│   ├── healthcheck/           # Standalone healthcheck binary (separate `main`)
│   ├── integration/           # Cross-cutting Postgres integration tests (require live DB)
│   ├── mail/                  # SMTP invitation email sender
│   ├── migrations/            # golang-migrate SQL migrations (numbered, up/down pairs)
│   ├── pb/                    # Generated gRPC/REST stubs from protos/url_service.proto (backend copy)
│   ├── shortener/             # gRPC service implementations (Shortener + Auth RPCs)
│   ├── main.go                # Backend process entry point
│   ├── config.json.example    # Default config file copied into container image
│   ├── Containerfile           # Multi-stage build → distroless image
│   └── go.mod                  # Module: github.com/incidrthreat/goshorten/backend
├── frontend/                  # Go edge server (SPA host + API proxy + redirect handler)
│   ├── cmd/                   # Frontend process entry point (`main.go`)
│   ├── healthcheck/           # Standalone healthcheck binary for this service
│   ├── pb/                    # Generated gRPC client stubs (frontend's own copy)
│   ├── ui-src/                # React/TypeScript SPA source (built to `dist/`, served by webapp)
│   │   └── src/
│   │       ├── api/            # `client.ts` — fetch wrapper for all backend REST calls
│   │       ├── components/     # Reusable UI components (Layout, TagInput, WorkspaceSwitcher)
│   │       ├── hooks/          # `useAuth.ts` — auth/session state hook
│   │       ├── pages/          # Route-level screens (one file per route)
│   │       │   └── admin/      # Platform-admin-only screens (Users, OIDCProviders)
│   │       ├── types/          # Shared TypeScript types (`index.ts`)
│   │       ├── App.tsx         # Route table (react-router-dom)
│   │       └── main.tsx        # React root mount
│   ├── webapp/                 # Frontend HTTP handler package (router, proxy, redirect logic)
│   ├── Containerfile
│   └── go.mod                  # Module: github.com/incidrthreat/goshorten/frontend
├── protos/                    # Proto contract (single source of truth for the API)
│   ├── url_service.proto      # Shortener + Auth service/message definitions
│   ├── google/api/             # Vendored google.api.http annotations
│   └── protoc-gen-openapiv2/   # Vendored OpenAPI annotation options
├── k8s/                       # Kubernetes manifests (one file per resource)
├── screenshots/               # README screenshots (static assets)
├── docker-compose.yml          # Local dev stack: grpcbackend, redis, postgres, frontend
├── generate-tls-certs.sh       # Local TLS cert generation helper
├── VERSION                     # Single-line version string, injected via ldflags at build time
└── .github/workflows/          # CI: backend, frontend, and React UI lint/build/test jobs
```

## Directory Purposes

**`backend/auth/`:**
- Purpose: All authentication and multi-tenancy logic
- Contains: `interceptor.go` (gRPC auth middleware), `store.go` (users/sessions, ~733 lines — largest file in the package), `workspace.go` (tenant CRUD), `membership.go` (workspace-role management), `invitation.go` (invite lifecycle + expiry sweep), `oidc.go` (OIDC provider registration/callback), `jwt.go` (token issue/verify), `password.go` (bcrypt hashing)
- Key files: `backend/auth/interceptor.go` (context accessor functions used throughout `shortener/`), `backend/auth/store.go`

**`backend/data/`:**
- Purpose: Persistence abstraction and implementations
- Contains: `store.go` (the `URLStore` interface + shared param/result structs), `postgres.go` (Postgres implementation + RLS transaction helpers `withWS`/`withBypass`), `cached_store.go` (Redis-caching decorator over any `URLStore`), `redis.go`, `analytics.go`, `tags.go`, `visit.go` (async click logger), `useragent.go`, `codegen.go` (short-code generation), `migrate.go` (migration runner)
- Key files: `backend/data/postgres.go` (largest, 698 lines — read this first for any storage change)

**`backend/gateway/`:**
- Purpose: HTTP-facing layer that is not itself gRPC — translates REST↔gRPC and hosts endpoints with no proto equivalent
- Contains: `gateway.go` (server bootstrap, CORS, health/ready endpoints, `grpc-gateway` mux registration), `admin.go` (1030 lines — admin + self-service auth REST routes: users, OIDC providers, API keys, sessions), `members.go` (581 lines — workspace membership/invitation REST routes), `workspace.go` (workspace CRUD REST routes), `swagger_ui.go` (serves Swagger UI at `/api/v1/docs`)
- Key files: `backend/gateway/admin.go` is the largest file in the whole backend — check here first for any REST-only endpoint

**`backend/shortener/`:**
- Purpose: gRPC service implementations (business logic layer)
- Contains: `shortener.go` (`CreateServer` — CRUD for short URLs, workspace-scoped lookups), `auth_service.go` (`AuthServer` — login, OIDC, sessions, API keys), `analytics.go`, `tags.go`, `qr.go` (QR code generation), `validate.go` (URL/slug validation rules)

**`backend/migrations/`:**
- Purpose: Versioned SQL schema changes, applied automatically at backend startup (`data.RunMigrations`)
- Naming: `NNNNNN_description.up.sql` / `.down.sql`, zero-padded sequential prefix (currently 000001–000009)
- Notable: `000007_multitenancy.up.sql` introduces workspaces; `000008_rls.up.sql` adds `FORCE ROW LEVEL SECURITY` policies keyed on `app.current_workspace_id`

**`backend/pb/` and `frontend/pb/`:**
- Purpose: Generated Go code from `protos/url_service.proto` (gRPC stubs, REST gateway glue, OpenAPI spec)
- Generated: Yes — do not hand-edit; regenerate from proto
- Committed: Yes (checked into git, not gitignored) — kept in sync manually across both trees

**`frontend/webapp/`:**
- Purpose: The frontend binary's only application-logic package
- Contains: `app.go` (`App` struct — SPA dir, backend URL, gRPC conn), `routes.go` (router: API proxy → redirect → SPA), `grpc.go` (`GetURL` redirect handler calling backend gRPC directly), `proxy.go` (reverse proxy to backend REST gateway)

**`frontend/ui-src/src/pages/`:**
- Purpose: One file per top-level route, matched 1:1 with routes declared in `App.tsx`
- Contains: `Dashboard.tsx`, `CreateURL.tsx`, `EditURL.tsx`, `URLDetail.tsx`, `Tags.tsx`, `APIKeys.tsx`, `Members.tsx`, `SettingsPage.tsx`, `Login.tsx`, `AcceptInvite.tsx`, `OIDCCallback.tsx`, `Preview.tsx` (public link preview), `Expired.tsx` (expired/invalid code landing page)
- `pages/admin/`: platform-admin-only screens gated by `user.role === 'admin'` in `App.tsx`

**`k8s/`:**
- Purpose: Raw Kubernetes manifests (no Helm/Kustomize), one file per logical resource group
- Contains: `namespace.yaml`, `backend.yaml` (Deployment + probes on `/healthz`), `frontend.yaml`, `postgres.yaml`, `redis.yaml`, `secret.yaml` (secret *template*, not real secrets)
- Generated: No
- Committed: Yes

**`protos/`:**
- Purpose: API contract definitions
- Contains: `url_service.proto` (the only project-owned proto — defines `Shortener` and `Auth` gRPC services with `google.api.http` REST bindings), `google/api/*.proto` and `protoc-gen-openapiv2/*.proto` (vendored dependencies needed by `protoc`/`buf` codegen, not project code)

## Key File Locations

**Entry Points:**
- `backend/main.go`: Backend process bootstrap (config, migrations, stores, gRPC+REST servers)
- `frontend/cmd/main.go`: Frontend process bootstrap (gRPC client, HTTP server)
- `backend/healthcheck/main.go`, `frontend/healthcheck/main.go`: Standalone probe binaries
- `frontend/ui-src/src/main.tsx`: React app mount point

**Configuration:**
- `backend/config/config.go`: Config struct definitions + env-var override methods (`Get*` methods on each conf struct)
- `backend/config.json.example`: Default config copied into the container image at build time; runtime env vars override it
- `docker-compose.yml`: Local dev topology and env var wiring
- `k8s/*.yaml`: Production topology; secrets sourced from `k8s/secret.yaml` / `Secret` objects

**Core Logic:**
- `backend/shortener/shortener.go`: URL CRUD RPCs
- `backend/shortener/auth_service.go`: Auth RPCs
- `backend/auth/interceptor.go`: Central authz/tenant-resolution logic
- `backend/data/postgres.go`: RLS-aware persistence, the most important file to read before any schema/query change

**Testing:**
- `backend/integration/isolation_test.go`, `backend/integration/membership_test.go`: Postgres-backed integration tests, gated on `GOSHORTEN_TEST_DSN` env var
- No `*_test.go` unit test files were found alongside package sources at exploration time — test coverage is concentrated in `backend/integration/`

## Naming Conventions

**Files:**
- Go: lowercase, snake-free single words or short compound names matching the primary type they define (e.g. `workspace.go` defines `Workspace`, `interceptor.go` defines `AuthInterceptor`) — one dominant exported type/concern per file
- Generated Go: `<proto_name>.pb.go`, `<proto_name>.pb.gw.go`, `<proto_name>_grpc.pb.go` (standard `protoc-gen-go`/`protoc-gen-go-grpc`/`protoc-gen-grpc-gateway` output naming)
- SQL migrations: `NNNNNN_description.{up,down}.sql`, zero-padded 6-digit sequence
- React/TypeScript: PascalCase for component/page files (`CreateURL.tsx`, `WorkspaceSwitcher.tsx`), camelCase for non-component modules (`client.ts`, `useAuth.ts`)

**Directories:**
- Go backend: lowercase single-word package directories named after their domain concern (`auth`, `data`, `gateway`, `shortener`, `mail`), each directory is exactly one Go package
- Frontend SPA: conventional React app layout — `pages/`, `components/`, `hooks/`, `api/`, `types/`, with `pages/admin/` as a nested sub-route grouping

## Where to Add New Code

**New gRPC-backed feature (URL/tag/analytics domain):**
- Proto definition: `protos/url_service.proto` (add RPC + messages, regenerate `backend/pb` and `frontend/pb`)
- Server implementation: add method to `backend/shortener/shortener.go` (or a new file in `backend/shortener/` for a distinct sub-domain, following the `tags.go`/`analytics.go`/`qr.go` split)
- Storage: extend the `URLStore` interface in `backend/data/store.go`, implement in `backend/data/postgres.go` inside a `withWS`/`withBypass` transaction, add a passthrough in `backend/data/cached_store.go` if caching applies
- Migration: new numbered file pair in `backend/migrations/`
- Frontend SPA: new page in `frontend/ui-src/src/pages/`, route entry in `frontend/ui-src/src/App.tsx`, API call added to `frontend/ui-src/src/api/client.ts`

**New REST-only admin/self-service endpoint (no proto):**
- Handler: add to `backend/gateway/admin.go` (auth/user/OIDC/API-key concerns) or `backend/gateway/members.go` (workspace membership/invitations) or `backend/gateway/workspace.go` (workspace CRUD), following the existing `requireAuth`/`requireAdmin`/`requireWorkspaceMember` guard pattern
- Registration: wire the new route in the handler's `Register(mux)` method, called from `backend/gateway/gateway.go`

**New auth/tenancy capability:**
- Store logic: `backend/auth/store.go` (users/sessions) or a new domain-specific file in `backend/auth/` (following `workspace.go`/`membership.go`/`invitation.go` pattern)
- Context propagation: add typed context key + accessor functions in `backend/auth/interceptor.go` if the new data needs to flow from auth to RPC handlers

**Utilities:**
- Backend shared helpers: co-locate within the consuming package (this codebase does not use a generic `backend/utils` or `backend/pkg` directory — e.g. `backend/data/useragent.go`, `backend/data/codegen.go` are single-purpose helper files inside their domain package)
- Frontend shared helpers: `frontend/ui-src/src/api/client.ts` for all backend calls; `frontend/ui-src/src/types/index.ts` for shared TS types

## Special Directories

**`backend/pb/` / `frontend/pb/`:**
- Purpose: Generated gRPC/REST/OpenAPI code from `protos/url_service.proto`
- Generated: Yes
- Committed: Yes (must be regenerated and committed manually when the proto changes — no build-time codegen step in `Containerfile`)

**`backend/migrations/`:**
- Purpose: Applied automatically on every backend startup via `data.RunMigrations` (`backend/data/migrate.go`) — not a manual ops step
- Generated: No (hand-written SQL)
- Committed: Yes

**`frontend/ui-src/` build output (`dist/`):**
- Purpose: Vite build output, served as static files by `frontend/webapp` (`GOSHORTEN_SPA_DIR` env var, default `./dist`)
- Generated: Yes (`npm run build` in `frontend/ui-src/`)
- Committed: No (build artifact, produced during `frontend/Containerfile` image build)

**`k8s/secret.yaml`:**
- Purpose: Template/example `Secret` manifest referenced by `backend.yaml`'s `secretKeyRef`s (`REDIS_PASSWORD`, `POSTGRES_*`, `JWT_SECRET`, `ADMIN_EMAIL`, `ADMIN_PASSWORD`)
- Generated: No
- Committed: Yes — treat as a template; do not put real production secret values in this file

---

*Structure analysis: 2026-08-31*
