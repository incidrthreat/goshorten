# Codebase Concerns

**Analysis Date:** 2026-08-31

## Tech Debt

**Fragile write-method detection for viewer role enforcement:**
- Issue: `isWriteMethod()` in `backend/auth/interceptor.go:225-231` gates read-only "viewer" workspace roles by substring-matching gRPC method names against a hardcoded verb list (`Create`, `Update`, `Delete`, `Rename`, `Revoke`, `Roll`, `Assign`). This works for the current proto surface (`protos/url_service.proto`) but is an allowlist-by-accident, not an explicit capability declaration.
- Files: `backend/auth/interceptor.go:225-231`, `protos/url_service.proto`
- Impact: Any future RPC named without one of those verbs (e.g. `SetActive`, `Toggle`, `Bulk*`, `Move`, `Merge`, `Publish`, `Archive`) will silently bypass the viewer read-only restriction and allow mutation. There is no test asserting the verb list stays in sync with the proto.
- Fix approach: Replace the substring heuristic with an explicit per-method `map[string]bool` (mirroring `PublicMethods`/`AdminMethods`), or add a proto option/annotation marking mutating RPCs, plus a test that fails when an RPC is added without being classified.

**Single 1030-line "kitchen sink" REST handler file:**
- Issue: `backend/gateway/admin.go` (1030 lines) mixes unrelated concerns: user management, URL admin overrides, password/account/preferences, sessions, OIDC provider CRUD, and system settings, all as methods on one `AdminHandler` struct.
- Files: `backend/gateway/admin.go`
- Impact: High change-collision risk (many unrelated features touch the same file), harder code review, and harder to reason about which handlers enforce which authorization level.
- Fix approach: Split into per-domain files (`admin_users.go`, `admin_oidc.go`, `admin_account.go`, `admin_settings.go`) sharing the existing `AdminHandler` receiver — no behavior change needed, pure file decomposition.

**Old, unmaintained Redis client:**
- Issue: `go.mod` pins `github.com/go-redis/redis v6.15.9+incompatible`, a major-version-behind client (current upstream is `github.com/redis/go-redis/v9`). The `+incompatible` marker means it predates Go modules properly.
- Files: `backend/go.mod`, `backend/data/redis.go`, `backend/data/cached_store.go`
- Impact: No connection pooling context support, no modern Redis Cluster/Sentinel features, and the package is effectively unmaintained upstream — future Redis server versions or TLS requirements may not be supported.
- Fix approach: Migrate to `github.com/redis/go-redis/v9`, updating the small `Cache *redis.Client` surface used in `cached_store.go` and `redis.go`.

**JWT platform role and account status are not re-validated per-request:**
- Issue: `backend/auth/interceptor.go` and `backend/gateway/admin.go:requireAuth` verify JWT signature/expiry and (for JWT) session revocation via `ValidateSession`, but never re-check `users.role` or `users.is_active` against the database on each request. `is_active` is only checked in the API-key path (`backend/auth/store.go:523`, inside `ValidateAPIKey`).
- Files: `backend/auth/interceptor.go:97-140`, `backend/gateway/admin.go:45-58`, `backend/auth/store.go:523`
- Impact: Demoting a platform admin to a regular user, or deactivating a user via `PATCH /api/v1/admin/users/{id}` (`backend/gateway/admin.go:204-262`, calling `AuthStore.UpdateUser`), does not revoke that user's existing sessions. The affected account keeps its old role/access for up to `token_expiry_hours` (default 24h, `backend/config/config.go:110-113`) unless an operator also explicitly revokes their sessions.
- Fix approach: Have `UpdateUser` revoke all sessions for the target user when `Role` or `IsActive` changes (there is already a `RevokeOtherSessions` primitive in `backend/auth/store.go:366` to model a full revoke-all from), or check `is_active`/role freshness in `ValidateSession`.

**Workspace shortcode namespace is global, not per-tenant:**
- Issue: `urls.code` carries a single global `UNIQUE` constraint (`backend/migrations/000001_initial_schema.up.sql:22`) that was never made workspace-scoped when multi-tenancy was added in migration `000007_multitenancy.up.sql`. Tags were explicitly made workspace-scoped (`idx_tags_workspace_name`, same migration) but codes were not.
- Files: `backend/migrations/000001_initial_schema.up.sql:22`, `backend/migrations/000007_multitenancy.up.sql`
- Impact: All tenants compete for the same finite short-code keyspace; a large or abusive tenant can exhaust desirable/short codes for every other tenant, and custom-slug collisions are cross-tenant (workspace A cannot use a slug workspace B already claimed, even though the two are otherwise fully isolated).
- Fix approach: Decide if this is an intentional product constraint (shared vanity-domain namespace) and document it, or migrate to a `(workspace_id, code)` unique constraint if per-tenant code space is desired — the latter is a breaking schema change requiring redirect-lookup changes since public redirects currently resolve by code alone.

## Known Bugs

**None identified as confirmed runtime bugs.** The concerns below are gaps/risks rather than reproduced defects; no `TODO`/`FIXME`/`HACK` markers exist in `backend/**/*.go` or `frontend/ui-src/src/**/*.{ts,tsx}` (verified via repo-wide grep), suggesting issues are either undocumented in-code or genuinely absent — treat the Fragile Areas and Test Coverage Gaps sections as the primary bug-risk surface.

## Security Considerations

**Tenant-isolation and membership integration tests never run in CI:**
- Risk: `backend/integration/isolation_test.go` (`TestWorkspaceIsolation`, `TestRowLevelSecurity`) and `backend/integration/membership_test.go` (`TestSignupCreatesOwnerMembership`, `TestLastOwnerInvariant`, `TestInvitationAcceptGrantsScopedAccess`, `TestTransferOwnership`) are the only tests validating the Phase 14/15 multi-tenancy security model. Each calls `testDSN(t)` which does `t.Skip("GOSHORTEN_TEST_DSN not set...")` when the env var is absent (`backend/integration/isolation_test.go:28-34`).
- Files: `.github/workflows/ci.yml`, `backend/integration/isolation_test.go`, `backend/integration/membership_test.go`
- Current mitigation: None. `.github/workflows/ci.yml` only runs `go test ./...` under `if: github.event_name == 'release'` (lines 44, 86) — i.e., never on push/PR — and even on release, no `GOSHORTEN_TEST_DSN` env var or Postgres service container is configured, so these tests self-skip unconditionally in CI today.
- Recommendations: Add a `postgres` service container to `ci.yml`, set `GOSHORTEN_TEST_DSN` for the backend job, and run `go test ./...` (including `./integration/...`) on every push/PR, not just releases. This is the single highest-value CI change given tenant isolation is the most security-sensitive recent addition.

**Login has no brute-force protection:**
- Risk: `AuthServer.Login` (`backend/shortener/auth_service.go:28`) checks credentials via bcrypt and logs the attempt (`AuthStore.LogSignIn`) but has no rate limiting, attempt counting, or lockout. No rate-limiting middleware or library exists anywhere in the backend (verified via repo-wide search for `ratelimit`/`limiter`).
- Files: `backend/shortener/auth_service.go:28-84`, `backend/gateway/gateway.go`
- Current mitigation: bcrypt cost 12 (`backend/auth/password.go:13`) slows individual guesses, but nothing prevents distributed/sustained credential-stuffing against `/Auth/Login`.
- Recommendations: Add per-IP and per-account rate limiting (e.g. token bucket in Redis, which is already a dependency) in front of `Login` and `OIDCCallback`.

**No allowlist on redirect-target URL schemes:**
- Risk: `NormalizeURL` (`backend/shortener/validate.go:8-38`) only checks that a parsed URL has a non-empty `Host`; it does not restrict `u.Scheme` to `http`/`https`. A long URL submitted as `javascript://payload` or `data://...` (containing `://` so the auto-`https://`-prefix step is skipped) passes validation and would be stored/served as the redirect target in the `Location` header.
- Files: `backend/shortener/validate.go:8-38`
- Current mitigation: None in this function; browsers generally refuse to auto-execute `javascript:`/`data:` from a `Location` header, which limits (but does not eliminate) practical exploitability, and some HTTP clients or embedded webviews may behave differently.
- Recommendations: Restrict accepted schemes to `http`/`https` (and any explicitly supported custom scheme for deep-linking) in `NormalizeURL`.

**JWT stored in `localStorage` (no httpOnly cookie option):**
- Risk: The frontend stores the auth token and active workspace id in `localStorage` (`frontend/ui-src/src/api/client.ts:10,20`) rather than an httpOnly cookie. Any XSS on the SPA origin can read and exfiltrate the token directly via JS.
- Files: `frontend/ui-src/src/api/client.ts:10,20`, `frontend/ui-src/src/hooks/useAuth.ts`
- Current mitigation: No `dangerouslySetInnerHTML`, `eval`, or `new Function` usage found anywhere in `frontend/ui-src/src` (reduces first-party XSS surface), and CORS is same-origin in typical deployment.
- Recommendations: If the SPA and API ever share a browsable origin with third-party content (embeds, user-supplied HTML rendering, etc.), migrate to httpOnly session cookies with CSRF protection instead of bearer-token-in-localStorage.

**Wildcard CORS on the REST/gRPC-gateway:**
- Risk: `corsMiddleware` (`backend/gateway/gateway.go:145-152`) sets `Access-Control-Allow-Origin: *` unconditionally and allows the `Authorization` header, meaning any origin can make credentialed-via-header API calls if it obtains a token.
- Files: `backend/gateway/gateway.go:145-152`
- Current mitigation: Bearer-token auth (not cookies) means the browser same-origin policy still prevents a third-party site from reading `localStorage`, so this is lower-severity than cookie-based wildcard CORS, but it does mean a leaked/phished token can be replayed from any origin including attacker-controlled pages running fetch on behalf of a tricked user.
- Recommendations: If the deployment model is single-tenant self-hosting with a known frontend origin, restrict `Access-Control-Allow-Origin` to `AppBaseURL` (already present in `Configuration.AppBaseURL`, `backend/config/config.go:65`).

**OIDC client secrets and invitation tokens stored as plaintext in Postgres:**
- Risk: `oidc_providers.client_secret` (`backend/migrations/000003_auth.up.sql:18`) and `invitations.token` are stored unencrypted in the database; anyone with DB read access (backup leak, SQL injection elsewhere, insider) obtains live OIDC client secrets and can also see/replay any still-pending invite token before the invitee does (`backend/auth/invitation.go` generates a 32-byte random token but stores it in cleartext, unlike API keys which store only a SHA-256 hash — see `backend/auth/password.go:38-41` vs `backend/auth/invitation.go`).
- Files: `backend/migrations/000003_auth.up.sql:18`, `backend/auth/invitation.go`, `backend/auth/store.go:590-627`
- Current mitigation: None beyond normal DB access controls.
- Recommendations: Store invitation tokens hashed (mirroring the API-key pattern: `HashAPIKey` in `backend/auth/password.go:38-41`) and compare hashes on accept; consider encrypting `client_secret` at rest or moving OIDC secrets to a secrets manager referenced by name.

**Default/example credentials committed in `docker-compose.yml`:**
- Risk: `docker-compose.yml` contains inline default values for `GOSHORTEN_POSTGRES_PASSWORD`, `GOSHORTEN_ADMIN_PASSWORD`, and `POSTGRES_PASSWORD` (values not reproduced here). This is standard for local-dev compose files but is a real risk if a deployer copies the compose file into production without changing them.
- Files: `docker-compose.yml`
- Current mitigation: These are clearly dev-oriented defaults; `k8s/` manifests should be checked separately for production deployment guidance.
- Recommendations: Add a README/compose comment making explicit "change these before any non-local use," or generate random values via a setup script.

## Performance Bottlenecks

**Per-request workspace/role resolution adds multiple round-trips to every authenticated call:**
- Problem: `AuthInterceptor.authorize` (`backend/auth/interceptor.go:97-140`) performs, for every JWT-authenticated gRPC call: (1) `ValidateSession` query, (2) `ResolveActiveWorkspace` (up to 3 sequential queries: header check, session lookup, default-workspace lookup — `backend/auth/workspace.go:200-231`), and (3) `WorkspaceRole` (another query, `backend/auth/workspace.go:143-166`, which itself may run a second fallback query). That is up to 5 sequential DB round-trips before the actual handler runs.
- Files: `backend/auth/interceptor.go:97-140`, `backend/auth/workspace.go:143-231`
- Cause: No caching layer for session validity, active-workspace resolution, or per-workspace role — every gRPC call re-derives them from Postgres even though sessions/roles change infrequently relative to request volume.
- Improvement path: Cache `(sessionID → valid)`, `(userID, sessionID → active workspace)`, and `(userID, workspaceID → role)` in Redis (already a dependency) with a short TTL and explicit invalidation on role change/session revoke/workspace switch, or embed the resolved workspace/role in the JWT itself (re-issued on switch) to cut this to a signature check.

**No pagination cap surfaced consistently; `List` endpoints rely on caller-provided page size:**
- Problem: `AuthStore.ListUsers` (`backend/auth/store.go:264-310`) clamps `pageSize` to 100, but the pattern is duplicated ad hoc across handlers rather than centralized, and `ListMembers` (`backend/auth/membership.go:78-96`) and `ListWorkspacesForUser` (`backend/auth/workspace.go:81-105`) have no pagination or row cap at all.
- Files: `backend/auth/membership.go:78-96`, `backend/auth/workspace.go:81-105`
- Cause: These were sized for the current low member/workspace-count use case (small teams) and never revisited as unbounded lists.
- Improvement path: Low priority today (membership/workspace counts are naturally small), but add limits if the product ever supports very large organizations (hundreds+ of members per workspace).

**Redis cache invalidation is best-effort with only warning logs:**
- Problem: Every cache write/invalidate in `backend/data/cached_store.go` (`Save`, `Load` backfill, `Create`, `Update`, `invalidate`) treats Redis errors as non-fatal, logging a warning and continuing (`log.Warn("Redis Cache", ...)`).
- Files: `backend/data/cached_store.go:24-29,50-56,79-85,101-113,145-149`
- Cause: Deliberate design so Redis outages degrade to Postgres-only rather than failing requests — reasonable for read-through caching, but stale cache entries after a failed `invalidate()` (e.g. `Update`, `backend/data/cached_store.go:96-113`) mean a URL edit/deactivation may still serve the old target from Redis until the natural TTL expires or the next cache-hit/primary-reject cycle (`CachedStore.Load`, `backend/data/cached_store.go:33-46`, does re-check Postgres on every cache hit as a partial mitigation).
- Improvement path: Already partially mitigated by the synchronous primary re-check on cache hit; if that check is ever removed for a performance win, invalidation failures would become a correctness bug, not just a performance one — worth a code comment/test pinning this behavior.

## Fragile Areas

**`backend/gateway/admin.go` (1030 lines, 20 handler methods):**
- Files: `backend/gateway/admin.go`
- Why fragile: Single file backing users, sessions, OIDC providers, account/preferences, and system settings REST endpoints. A change to one domain (e.g. OIDC provider validation) risks merge conflicts and accidental scope creep into unrelated handlers, and it is easy to add a new endpoint without matching the authorization pattern used by its neighbors (`requireAuth` vs `requireAdmin` vs the workspace-scoped `requireWorkspaceMember`/`requireWorkspaceManager` in `backend/gateway/members.go`).
- Safe modification: Before editing, identify which of the three auth helper tiers (`requireAuth`, `requireAdmin`, or the workspace-scoped helpers in `members.go`) is appropriate for the new/changed endpoint, and grep for the existing sibling handler for the same resource type as a template.
- Test coverage: No unit tests exist for any handler in this file (see Test Coverage Gaps).

**Role/permission resolution split across three files with subtly different fallback logic:**
- Files: `backend/auth/interceptor.go`, `backend/auth/workspace.go`, `backend/gateway/members.go`
- Why fragile: Platform-level role (`admin`/`user`, from JWT `Claims.Role`) and workspace-level role (`owner`/`admin`/`member`/`viewer`, from `WorkspaceRole`) are two independent authorization axes that get merged differently in different call sites: `requireWorkspaceMember` in `backend/gateway/members.go:27-53` gives platform admins an implicit `RoleAdmin`-equivalent workspace role, while the gRPC interceptor path (`backend/auth/interceptor.go:113-133`) does not apply that same override for JWT-authenticated gRPC calls (a platform admin with no workspace membership and no default-workspace bridge gets `wsRole == ""`, which is treated as no access for `RoleViewer`-gated write checks — effectively platform admins may be blocked from writing to workspaces they don't belong to via gRPC even though the REST membership endpoints grant them an admin override).
- Safe modification: When touching authorization logic, check both `backend/auth/interceptor.go` (gRPC path, covers `Shortener`/`Auth` services) and `backend/gateway/members.go`/`workspace.go` (REST-only endpoints) — they are not backed by one shared authorization function.
- Test coverage: `backend/integration/membership_test.go` covers ownership/invitation invariants but does not test the platform-admin-crosses-workspace-boundary scenario described above, and per the CI concern above, does not run automatically anyway.

**RLS is enabled only on `urls`, not `clicks`/`tags`/`api_keys`/`domains`:**
- Files: `backend/migrations/000008_rls.up.sql`, `backend/data/postgres.go`
- Why fragile: The migration's own comment documents this as intentional (those tables are read in pre-workspace/global contexts like async click logging and API-key validation where a hard RLS gate would break the flow), but it means `clicks`, `tags`, `api_keys`, and `domains` have **no database-level backstop** if a future query in `backend/data/*.go` forgets a `WHERE workspace_id = $N` clause — only `urls` is protected by `FORCE ROW LEVEL SECURITY`.
- Safe modification: Any new query touching `clicks`, `tags`, `api_keys`, or `domains` must be manually verified to include workspace scoping; there is no automatic enforcement to catch an omission.
- Test coverage: `TestRowLevelSecurity` in `backend/integration/isolation_test.go:149` only exercises the `urls` table's RLS policy.

## Scaling Limits

**Global short-code keyspace shared across all tenants:**
- Current capacity: Bounded by `VARCHAR(10)` code length (`backend/migrations/000001_initial_schema.up.sql:22`) shared across every workspace on the instance (see Tech Debt above).
- Limit: As tenant count and URL volume grow, desirable short/custom codes become scarce faster than a per-tenant model would allow, and custom-slug collisions become a cross-tenant support issue.
- Scaling path: Evaluate moving to `(workspace_id, code)` uniqueness if the product intends to scale to many independent tenants; requires changing the public redirect lookup (`backend/data/postgres.go:Load`/`Get`) to resolve by domain+code or similar since code alone would no longer be globally unique.

**Unbounded `clicks`/`orphan_visits` growth with IP addresses retained indefinitely:**
- Current capacity: `clicks` (`backend/data/visit.go:148-151`) and `orphan_visits` (`backend/data/visit.go:113-115`) store raw `ip_address`, `user_agent`, `referer`, `country`, `city` per visit with no retention/TTL policy or anonymization step visible in the migrations or store code.
- Limit: Table growth is unbounded and proportional to redirect traffic; no partitioning, archival, or row-expiry job was found (`backend/data/analytics.go`, `backend/data/visit.go`). This is also a data-retention/privacy consideration (raw IPs kept indefinitely) beyond pure scaling.
- Scaling path: Add a retention policy (scheduled deletion or anonymization of `ip_address` after N days) and consider partitioning `clicks` by time if visit volume grows significantly; add this as a system-settings-configurable retention window given `system_settings` (migration `000006_system_settings.up.sql`) already exists as a pattern for instance-level config.

## Dependencies at Risk

**`github.com/go-redis/redis v6.15.9+incompatible`:**
- Risk: Major-version-behind (current is `github.com/redis/go-redis/v9`), no longer receiving updates under this import path.
- Impact: `backend/data/redis.go`, `backend/data/cached_store.go` — the redirect cache and Load/Save fast path depend on this client.
- Migration plan: Swap to `github.com/redis/go-redis/v9`; API surface used here (`Get`, `Set`, `Del`) maps closely to v9's context-aware equivalents (`GetContext`, `SetContext`, `DelContext`), so the change is localized to `backend/data/redis.go` and `backend/data/cached_store.go`.

## Missing Critical Features

**No CI enforcement of the multi-tenancy security test suite:**
- Problem: Covered in detail under Security Considerations above — `isolation_test.go` and `membership_test.go` never execute in CI (neither on PR, due to the `release`-only gate in `.github/workflows/ci.yml`, nor on release, due to the missing `GOSHORTEN_TEST_DSN`/Postgres service).
- Blocks: Confidence that a future change cannot silently regress tenant isolation or the last-owner/membership invariants; currently this can only be caught by a developer manually running `GOSHORTEN_TEST_DSN=... go test ./integration/...` locally.

**No frontend test suite or tooling:**
- Problem: `frontend/ui-src/package.json` has no test runner (no Jest/Vitest/Testing Library) in `devDependencies`, and no `*.test.*`/`*.spec.*` files exist anywhere under `frontend/ui-src/src`.
- Blocks: Any regression testing of authentication flows, workspace switching, member/invitation UI, or form validation logic in the 24 TS/TSX source files (5,374 total lines) is entirely manual.

## Test Coverage Gaps

**No unit tests for any Go package except two integration test files:**
- What's not tested: `backend/auth/*.go` (interceptor, jwt, password, oidc, workspace, membership, invitation, store — 733-line `store.go` has zero unit tests), `backend/gateway/*.go` (admin.go, members.go, workspace.go, gateway.go — 1030+581+295+160 lines with zero tests), `backend/shortener/*.go` (auth_service.go, shortener.go, tags.go, analytics.go, qr.go, validate.go — zero tests despite `validate.go` containing the URL-scheme validation gap noted above), `backend/data/*.go` (postgres.go, cached_store.go, redis.go, analytics.go, visit.go, tags.go — zero tests).
- Files: All of `backend/auth/`, `backend/gateway/`, `backend/shortener/`, `backend/data/` except the two files in `backend/integration/`.
- Risk: Regressions in JWT verification, password hashing, RLS-transaction wrapping (`backend/data/postgres.go:withWS`/`withBypass`), or slug/URL validation would only surface via manual QA or in production.
- Priority: High — `backend/auth/interceptor.go` and `backend/data/postgres.go` (RLS transaction scoping) are the two files most directly responsible for tenant isolation and warrant unit tests independent of the Postgres-dependent integration suite.

**No frontend tests at all:**
- What's not tested: Every component/page/hook under `frontend/ui-src/src` (`useAuth.ts`, `client.ts`, all pages in `frontend/ui-src/src/pages/`).
- Files: `frontend/ui-src/src/**`
- Risk: Auth-token handling (`frontend/ui-src/src/api/client.ts`), workspace-switch UI (`frontend/ui-src/src/components/WorkspaceSwitcher.tsx`), and invite-accept flow (`frontend/ui-src/src/pages/AcceptInvite.tsx`) can regress silently; `tsc --noEmit` in CI (`.github/workflows/ci.yml`) only catches type errors, not behavioral regressions.
- Priority: Medium — lower than the backend tenant-isolation gap, but the invite-accept and workspace-switch flows are the newest (Phase 15) and least battle-tested UI paths.

**No global 401/session-expiry handling in the frontend:**
- What's not tested (and not implemented): `frontend/ui-src/src/api/client.ts:9-38` throws a typed `APIError` on non-OK responses, but no page (`grep` across `frontend/ui-src/src/pages/*.tsx` for `APIError`/`status === 401` found zero matches) catches a 401 to force logout/redirect-to-login. `useAuth.ts` only validates the token once on mount (`checkAuth` in the initial `useEffect`, `frontend/ui-src/src/hooks/useAuth.ts:38-40`).
- Files: `frontend/ui-src/src/api/client.ts`, `frontend/ui-src/src/hooks/useAuth.ts`, all files under `frontend/ui-src/src/pages/`
- Risk: A user whose token expires or is revoked mid-session (e.g. an admin revokes their session, or the 24h JWT expires while the tab is open) will see ad hoc, per-page error states rather than being redirected to `/login`, since no shared fetch-interceptor or React context enforces this.
- Priority: Medium — a UX gap more than a security hole (the backend still rejects the request correctly), but worth fixing alongside any future work on `client.ts`.

---

*Concerns audit: 2026-08-31*
