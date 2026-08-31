# Testing Patterns

**Analysis Date:** 2026-08-31

**Overall state: test coverage is minimal and mostly aspirational.** `README.md` lists "Phase 9: Testing" as an *unchecked* roadmap item (`README.md:278-283`, includes unit tests for the repository/codegen/validation layers, integration tests via testcontainers-go, gRPC endpoint tests, REST e2e tests, and load/benchmark tests — none of these are implemented yet). The only tests that exist today are two Go integration test files covering multi-tenancy isolation and membership invariants added ad hoc alongside Phase 14/15 feature work. There is no frontend test suite at all.

## Test Framework

**Backend (Go):**
- Runner: standard library `testing` package (no testify, no ginkgo)
- No test config file; behavior is controlled entirely by env var `GOSHORTEN_TEST_DSN`
- Location: `backend/integration/` (its own Go package, separate from the code it tests)

**Run Commands:**
```bash
# From backend/ — runs all Go tests (integration tests self-skip without a DSN)
go test ./...

# To actually execute the Postgres-backed integration tests:
GOSHORTEN_TEST_DSN="postgres://goshorten:goshorten_secret@localhost:5432/goshorten?sslmode=disable" \
    go test ./integration/...
```
CI (`.github/workflows/ci.yml`) only runs `go test ./...` on `release` events (`if: github.event_name == 'release'`) — tests are **not** run on every push/PR, only lint (`golangci-lint-action`) and build/type-check run unconditionally.

**Frontend (Go REST gateway, `frontend/`):** same pattern (`go test ./...` in CI, release-gated), but no test files exist under `frontend/` (`find frontend -name "*_test.go"` returns nothing).

**Frontend (React/TypeScript, `frontend/ui-src/`):** **no test runner configured.** `package.json` has no `test` script and no `vitest`/`jest`/`@testing-library/*` dependency. CI's `ui` job only runs `npx tsc --noEmit` and `npm run build` (`.github/workflows/ci.yml`) — type-checking is the sole automated quality gate for the UI.

## Test File Organization

**Location:** Go integration tests live in a dedicated top-level package `backend/integration/`, not co-located with the code under test. There are no unit test files (`*_test.go`) inside `backend/auth/`, `backend/data/`, `backend/shortener/`, or `backend/gateway/` — when adding unit tests for those packages, co-locate them as `<file>_test.go` in the same package following standard Go convention (this repo has not yet established that pattern, so use idiomatic Go defaults: `package auth`, `TestXxx(t *testing.T)`).

**Naming:** `<feature>_test.go` — `backend/integration/isolation_test.go` (workspace data isolation), `backend/integration/membership_test.go` (ownership/membership invariants).

**Structure:**
```
backend/
  integration/
    isolation_test.go     # cross-tenant data-leak assertions (196 lines)
    membership_test.go    # owner/membership invariants (136 lines)
```

## Test Structure

**Suite organization:** flat `func TestXxx(t *testing.T)` functions, no subtests/table-driven pattern used yet. Each test is self-contained: call a shared `setup(t)` helper, create fixtures inline, assert, rely on `t.Cleanup` for teardown.

```go
func TestWorkspaceIsolation(t *testing.T) {
	pgStore, authStore, _ := setup(t)
	defer pgStore.Pool.Close()

	userA := makeUser(t, authStore, "alpha")
	userB := makeUser(t, authStore, "beta")
	wsA := *userA.DefaultWorkspaceID
	wsB := *userB.DefaultWorkspaceID

	recA, err := pgStore.Create(data.URLCreateParams{
		LongURL: "https://example.com/a", RedirectType: 302, IsCrawlable: true,
		UserID: &userA.ID, WorkspaceID: wsA,
	})
	if err != nil {
		t.Fatalf("create url A: %v", err)
	}
	// ... more setup, then assertions comparing workspace A vs B state
}
```
(`backend/integration/isolation_test.go:69-90`)

**Skip guard (test entry point):** every test starts by calling `testDSN(t)` (via `setup(t)`), which calls `t.Skip(...)` if `GOSHORTEN_TEST_DSN` is unset — this makes the whole integration suite opt-in and safe to run in environments without Postgres:
```go
func testDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("GOSHORTEN_TEST_DSN")
	if dsn == "" {
		t.Skip("GOSHORTEN_TEST_DSN not set; skipping Postgres integration test")
	}
	return dsn
}
```
(`backend/integration/isolation_test.go:29-35`)

**Setup pattern:** `setup(t)` runs real migrations against the target DSN and returns live store instances — tests exercise the actual `data.PostgresStore` / `auth.AuthStore` against a real Postgres, never a mock:
```go
func setup(t *testing.T) (*data.PostgresStore, *auth.AuthStore, *pgxpool.Pool) {
	t.Helper()
	dsn := testDSN(t)
	if err := data.RunMigrations(dsn, "../migrations"); err != nil {
		t.Fatalf("migrations: %v", err)
	}
	pgStore, err := data.NewPostgresStore(dsn, 4)
	if err != nil {
		t.Fatalf("new postgres store: %v", err)
	}
	authStore := auth.NewAuthStore(pgStore.Pool)
	return pgStore, authStore, pgStore.Pool
}
```
(`backend/integration/isolation_test.go:37-49`)

**Teardown pattern:** fixture-creation helpers register their own cleanup via `t.Cleanup(...)` at creation time rather than deferring in the test body:
```go
func makeUser(t *testing.T, store *auth.AuthStore, tag string) *auth.User {
	t.Helper()
	email := fmt.Sprintf("iso-%s-%d@example.test", tag, time.Now().UnixNano())
	u, err := store.CreateUser(context.Background(), email, tag, "user", nil, nil, nil)
	if err != nil {
		t.Fatalf("create user %s: %v", tag, err)
	}
	t.Cleanup(func() { _ = store.DeleteUser(context.Background(), u.ID) })
	return u
}
```
(`backend/integration/isolation_test.go:53-65`)
The connection pool itself is closed with a plain `defer pgStore.Pool.Close()` at the top of each `TestXxx` function (not via `t.Cleanup`), so keep this pattern when adding new integration tests: `t.Cleanup` for per-fixture teardown, `defer` for the pool itself.

**Assertion pattern:** no assertion library — plain `if` + `t.Fatalf("<context>: %v", err)` / `t.Fatalf("<context>: got %v, want %v", got, want)`. Error-type assertions use stdlib `errors.Is`:
```go
if err := authStore.UpdateMemberRole(ctx, ws, owner.ID, auth.RoleAdmin); !errors.Is(err, auth.ErrLastOwner) {
	t.Fatalf("demoting last owner: got %v, want ErrLastOwner", err)
}
```
(`backend/integration/membership_test.go:37-39`)

**Test data uniqueness:** emails/identifiers are made unique per run with `time.Now().UnixNano()` suffixes rather than a fixture/factory library: `fmt.Sprintf("iso-%s-%d@example.test", tag, time.Now().UnixNano())` (`backend/integration/isolation_test.go:55`).

## Mocking

**None used.** No mocking framework (no `gomock`, no `testify/mock`) is present. All existing tests run against a real Postgres instance rather than mocks/stubs. If a future unit-test layer is added for the gRPC service layer (`backend/shortener/shortener.go`, `backend/shortener/auth_service.go`), note that `data.URLStore` is defined as an interface (`backend/data/store.go`) specifically to allow substituting a fake/in-memory implementation in tests — this is the natural seam to mock against, but no such fake currently exists in the codebase.

## Fixtures and Factories

No fixture/factory files or directories exist. Test data is constructed inline per-test via small helper functions in `backend/integration/isolation_test.go` (`makeUser`), reused across the package's other test file (`membership_test.go`) since both live in `package integration`.

## Coverage

**Requirements:** none enforced. No coverage tooling configured in CI or `Makefile` (no `Makefile` present).

**View coverage (manual, not wired into CI):**
```bash
cd backend
go test -cover ./...
```

## Test Types

**Unit tests:** none present for `backend/auth`, `backend/data`, `backend/shortener`, `backend/gateway`, or `frontend/ui-src`. This is the largest coverage gap — see CONCERNS.md.

**Integration tests:** `backend/integration/*_test.go` — real-Postgres, opt-in via `GOSHORTEN_TEST_DSN`, focused specifically on multi-tenant data isolation (`isolation_test.go`) and membership/ownership invariants (`membership_test.go`). These intentionally run against the app's non-superuser DB role so Postgres Row-Level Security policies (migration `000008`, `backend/migrations/`) are exercised — a superuser DSN causes the RLS-specific assertions to self-skip (see package doc comment, `backend/integration/isolation_test.go:1-11`).

**E2E tests:** not used. No Playwright/Cypress/similar configured for the React UI.

## Common Patterns

**Skipping when infra unavailable:**
```go
if dsn == "" {
	t.Skip("GOSHORTEN_TEST_DSN not set; skipping Postgres integration test")
}
```

**Error-sentinel testing:**
```go
if err := authStore.RemoveMember(ctx, ws, owner.ID); !errors.Is(err, auth.ErrLastOwner) {
	t.Fatalf("removing last owner: got %v, want ErrLastOwner", err)
}
```

**Multi-tenant negative assertions** (verifying isolation/absence rather than presence) — the dominant pattern in `isolation_test.go`:
```go
for _, u := range listA.URLs {
	if u.Code == recB.Code {
		t.Fatalf("workspace A leaked workspace B's URL %q", recB.Code)
	}
	if u.WorkspaceID != wsA {
		t.Fatalf("list A returned a URL from workspace %d", u.WorkspaceID)
	}
}
```
(`backend/integration/isolation_test.go:96-103`)

When adding new backend tests, follow this same structure: real dependencies over mocks, `t.Skip` for missing infra, `t.Helper()` on all helper funcs, `t.Cleanup` for fixture teardown, plain stdlib assertions with descriptive `t.Fatalf` messages that name the operation being asserted.

---

*Testing analysis: 2026-08-31*
