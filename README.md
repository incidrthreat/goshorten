# GoShorten

This project spawned due to my curiousity in gRPC, golang, and building something practical to share.
___________________________
## What is GoShorten?
GoShorten is a self-hosted URL Shortener written in Golang.  It uses a gRPC server on the "backend" for API calls and stores data in a Redis Database.  The current Time-To-Live for each URL/Code is possible via the webgui.  Options for 5 min, 24 hrs, and 48 hrs are available.
___________________________
## Getting Started

### Prerequisites
Either container runtime works:
- [Docker](https://docs.docker.com/get-docker/) + [Docker Compose](https://docs.docker.com/compose/install/)
- [Podman](https://podman.io/getting-started/installation) + [Podman Compose](https://github.com/containers/podman-compose)

### How to Run GoShorten:
1. `git clone https://github.com/incidrthreat/goshorten.git`

2. `cd goshorten`

3. **Docker:**
   ```bash
   docker compose up -d
   ```
   **Podman:**
   ```bash
   podman compose up -d
   ```
   > If `podman compose` is not available, install the Python wrapper: `pip install podman-compose` and use `podman-compose up -d`

4. Open your browser to `http://localhost:8081` (legacy UI) or `http://localhost:8081/app` (React dashboard)

5. Login with `admin@goshorten.local` / `admin` (break-glass admin account)

### Configuration

All env vars have sensible defaults and work out of the box for local development. **None are mandatory.** For production, you should change the four security-sensitive values (marked with ⚠ below).

| Variable | Default | Description |
|----------|---------|-------------|
| `GOSHORTEN_REDIS_HOST` | `redis:6379` | Redis host:port |
| `GOSHORTEN_REDIS_PASS` | `mysecretpassword` | ⚠ Redis password |
| `GOSHORTEN_POSTGRES_HOST` | `postgres` | Postgres hostname |
| `GOSHORTEN_POSTGRES_PORT` | `5432` | Postgres port |
| `GOSHORTEN_POSTGRES_USER` | `goshorten` | Postgres user |
| `GOSHORTEN_POSTGRES_PASSWORD` | `goshorten_secret` | ⚠ Postgres password |
| `GOSHORTEN_POSTGRES_DB` | `goshorten` | Postgres database |
| `GOSHORTEN_JWT_SECRET` | `change-me-to-a-random-secret-in-production` | ⚠ JWT signing secret |
| `GOSHORTEN_ADMIN_EMAIL` | `admin@goshorten.local` | Break-glass admin email |
| `GOSHORTEN_ADMIN_PASSWORD` | `admin` | ⚠ Break-glass admin password |
| `GOSHORTEN_GRPC_HOST` | `grpcbackend:9000` | gRPC listen address |
| `GOSHORTEN_GATEWAY_ADDR` | `:8080` | REST gateway listen address |
| `GOSHORTEN_BACKEND_URL` | `http://grpcbackend:8080` | Frontend → backend proxy URL |
| `GOSHORTEN_GRPC_ADDR` | `grpcbackend:9000` | Frontend → gRPC backend address |

#### Setting Environment Variables

You have several options for passing env vars to the containers without creating them on your host machine:

**Option 1: `.env` file (recommended)**

Create a `.env` file in the project root next to `docker-compose.yml`. Both Docker Compose and Podman Compose automatically load it:

```bash
# .env
GOSHORTEN_REDIS_PASS=a-strong-redis-password
GOSHORTEN_POSTGRES_PASSWORD=a-strong-pg-password
GOSHORTEN_JWT_SECRET=replace-with-a-64-char-random-string
GOSHORTEN_ADMIN_PASSWORD=a-strong-admin-password
```

Then reference them in `docker-compose.yml` under `environment:`:
```yaml
environment:
  - GOSHORTEN_REDIS_PASS=${GOSHORTEN_REDIS_PASS}
  - GOSHORTEN_POSTGRES_PASSWORD=${GOSHORTEN_POSTGRES_PASSWORD}
```

**Option 2: `env_file` directive in compose**

Create a file (e.g., `goshorten.env`) with your overrides and point the compose services at it:

```yaml
# docker-compose.override.yml
services:
  grpcbackend:
    env_file:
      - goshorten.env
  frontend:
    env_file:
      - goshorten.env
```

Then run as usual — the override file is picked up automatically.

**Option 3: Inline on the command line (one-off / CI)**

```bash
# Docker
GOSHORTEN_JWT_SECRET=mysecret docker compose up -d

# Podman
GOSHORTEN_JWT_SECRET=mysecret podman compose up -d
```

**Option 4: Edit `docker-compose.yml` directly**

Hardcode values in the `environment:` block of each service. Simple but not recommended for secrets since the file is committed to version control.

> **Tip:** Add `.env` and `*.env` to your `.gitignore` to avoid committing secrets.

Note: To view statistics of a code in the legacy UI, append a `+` to the end of the URI.

Note: To view statistics of your code, append a `+` to the end of the uri.

___________________________
## Screenshots
#### Home Page
![Home Page](/screenshots/homepageV1.0.2.png)
#### Successful Code creation
![Success!](/screenshots/successfulcodeV1.0.2.png)
#### Invalid Code retreival
![Invalid](/screenshots/invalidcodeV1.0.2.png)
#### Statistics of generated code
![Invalid](/screenshots/successfulStatsV1.0.4.PNG)
__________________________
## Contributing

If you are interested in contributing to this project please send an email to `incidrthreat@hackmethod.com` or submit a PR with any changes you'd like to see.  If you run into issues please submit an issues "ticket" [here](https://github.com/incidrthreat/goshorten/issues).
___________________________
## Authors/Contributors

* *Initial* - [Incidrthreat](https://twitter.com/incidrthreat)

Update Todos

Phase 1: Foundation — Database & Storage Layer

1.1 Design Postgres schema (urls, clicks, api_keys, tags, users)

1.2 Add Postgres service to docker-compose.yml

1.3 Integrate a Go migration tool (golang-migrate or goose)

1.4 Write initial migration files (up/down)

1.5 Build Postgres repository layer (CRUD for URLs, clicks, tags, users)

1.6 Refactor Redis to cache-only role (read-through cache for redirects)

1.7 Implement cache invalidation strategy (write-through on create/update/delete)

Phase 2: Core Feature Parity with Shlink

2.1 Custom slugs / vanity URLs (user-provided short codes)

2.2 Configurable TTL (arbitrary expiration or no-expiry)

2.3 URL validation and normalization (scheme, trailing slash, IDN)

2.4 Crawlable/non-crawlable toggle (X-Robots-Tag, redirect type 301 vs 302)

2.5 Max visits limit (auto-disable after N clicks)

2.6 URL update/edit support (change target URL without changing code)

2.7 URL soft-delete and disable/enable toggle

2.8 QR code generation for short URLs

2.9 Multi-domain support (resolve different domains to different base URLs)

Phase 3: Authentication & Authorization

3.1 API key model (generate, revoke, scope per key)

3.2 gRPC interceptor for API key auth

3.3 Role-based access (admin vs regular user)

3.4 Rate limiting per API key (token bucket in Redis)

Phase 4: REST API Gateway

4.1 Add grpc-gateway annotations to proto definitions

4.2 Generate REST reverse proxy from protos

4.3 REST endpoints: POST /short-urls, GET /short-urls, GET /short-urls/{code}, PATCH, DELETE

4.4 REST endpoints: GET /short-urls/{code}/visits (analytics)

4.5 REST endpoint: GET /short-urls/{code}/qr-code

4.6 Pagination, filtering, and sorting on list endpoints

4.7 OpenAPI/Swagger spec generation from protos

4.8 API versioning strategy (v1 prefix)

Phase 5: Analytics & Visit Tracking

5.1 Capture full visit data (referrer, user-agent, IP, timestamp)

5.2 GeoIP lookup (MaxMind GeoLite2 — country, city)

5.3 Device/browser/OS parsing from user-agent

5.4 Async visit logging (channel/worker to avoid blocking redirects)

5.5 Aggregation queries (visits by day, by referrer, by country, by browser)

5.6 Bot/crawler detection and filtering from stats

5.7 Orphan visit tracking (visits to invalid/expired codes)

Phase 6: Tag System

6.1 Tag CRUD (create, rename, delete tags)

6.2 Many-to-many URL-tag relationship

6.3 Filter/search URLs by tag

6.4 Tag-level aggregated stats

Phase 7: Frontend Rebuild

7.1 Replace Go templates with a modern SPA (React, Svelte, or HTMX)

7.2 Dashboard: list all short URLs with search/filter/sort

7.3 Analytics dashboard with charts (visits over time, geo map, top referrers)

7.4 URL creation form (custom slug, TTL, tags, max visits, domain)

7.5 API key management UI

7.6 Settings/config page

Phase 8: Infrastructure & Operations

8.1 Structured JSON logging with log levels

8.2 Health check endpoints (liveness + readiness)

8.3 Prometheus metrics (request latency, cache hit/miss ratio, active URLs)

8.4 Graceful shutdown (drain connections, flush writes)

8.5 Configuration via env vars (12-factor app) alongside config file

8.6 CI pipeline (lint, test, build, docker image)

8.7 Dockerfile optimization (multi-stage, scratch/distroless base)

Phase 9: Testing

9.1 Unit tests for repository layer, code generation, validation

9.2 Integration tests against real Postgres + Redis (testcontainers-go)

9.3 gRPC endpoint tests (mock store, verify request/response)

9.4 REST API end-to-end tests

9.5 Load/benchmark tests for redirect path (vegeta or k6)

Phase 10: Import & CLI

10.1 Shlink-compatible import (read Shlink DB or API export)

10.2 CSV/JSON bulk import

10.3 CLI tool for admin operations (create URL, list, stats, manage keys)
Here's the full roadmap broken into 10 phases. The ordering is intentional — each phase builds on the one before it:

Phase 1: Foundation (Database & Storage)
Get Postgres in, migrate off Redis-as-primary. Everything else depends on this.

Phase 2: Core Feature Parity
The features Shlink users expect out of the box — custom slugs, flexible TTL, max visits, edit/disable URLs, QR codes, multi-domain.

Phase 3: Authentication & Authorization
API keys, role-based access, rate limiting. Required before exposing a public API.

Phase 4: REST API Gateway
Add grpc-gateway annotations to your existing protos to auto-generate a REST API. Include OpenAPI spec generation so consumers get docs for free.

Phase 5: Analytics & Visit Tracking
The big differentiator — full visit capture (referrer, geo, device), async logging so redirects stay fast, and aggregation queries in Postgres.

Phase 6: Tag System
Organize and filter URLs by tags, with tag-level aggregated stats.

Phase 7: Frontend Rebuild
Replace Bootstrap 4 templates with a proper SPA — dashboard, analytics charts, URL management, API key admin.

Phase 8: Infrastructure & Operations
Health checks, Prometheus metrics, graceful shutdown, env-var config, CI pipeline, optimized Docker images.

Phase 9: Testing
Unit, integration (testcontainers-go), gRPC, REST e2e, and load/benchmark tests on the redirect hot path.

Phase 10: Import & CLI
Shlink-compatible import for migration, bulk CSV/JSON import, and a CLI admin tool.