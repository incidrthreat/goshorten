// Package integration holds cross-cutting tests that need a real Postgres.
//
// These tests are skipped unless GOSHORTEN_TEST_DSN points at a Postgres instance
// (e.g. the one from docker-compose). Example:
//
//	GOSHORTEN_TEST_DSN="postgres://goshorten:goshorten_secret@localhost:5432/goshorten?sslmode=disable" \
//	    go test ./integration/...
//
// Run the app's containerized Postgres as a non-superuser role so the RLS backstop
// (migration 000008) is exercised; a superuser DSN bypasses RLS and that portion of
// the test self-skips.
package integration

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/incidrthreat/goshorten/backend/auth"
	"github.com/incidrthreat/goshorten/backend/data"
	"github.com/jackc/pgx/v5/pgxpool"
)

func testDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("GOSHORTEN_TEST_DSN")
	if dsn == "" {
		t.Skip("GOSHORTEN_TEST_DSN not set; skipping Postgres integration test")
	}
	return dsn
}

// setup runs migrations and returns the stores plus a connection pool.
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

// makeUser creates a user (and their auto-provisioned personal workspace).
func makeUser(t *testing.T, store *auth.AuthStore, tag string) *auth.User {
	t.Helper()
	email := fmt.Sprintf("iso-%s-%d@example.test", tag, time.Now().UnixNano())
	u, err := store.CreateUser(context.Background(), email, tag, "user", nil, nil, nil)
	if err != nil {
		t.Fatalf("create user %s: %v", tag, err)
	}
	if u.DefaultWorkspaceID == nil || *u.DefaultWorkspaceID == 0 {
		t.Fatalf("user %s has no auto-created workspace", tag)
	}
	t.Cleanup(func() { _ = store.DeleteUser(context.Background(), u.ID) })
	return u
}

// TestWorkspaceIsolation verifies that no workspace can read or mutate another's
// URLs through the repository layer (Phase 14.10).
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
	recB, err := pgStore.Create(data.URLCreateParams{
		LongURL: "https://example.com/b", RedirectType: 302, IsCrawlable: true,
		UserID: &userB.ID, WorkspaceID: wsB,
	})
	if err != nil {
		t.Fatalf("create url B: %v", err)
	}

	// List in workspace A must never include workspace B's URL.
	listA, err := pgStore.List(data.URLListParams{WorkspaceID: wsA, PageSize: 100})
	if err != nil {
		t.Fatalf("list A: %v", err)
	}
	for _, u := range listA.URLs {
		if u.Code == recB.Code {
			t.Fatalf("workspace A leaked workspace B's URL %q", recB.Code)
		}
		if u.WorkspaceID != wsA {
			t.Fatalf("list A returned a URL from workspace %d", u.WorkspaceID)
		}
	}

	// Get is global by code, but carries WorkspaceID so the service layer can reject
	// cross-tenant reads. Confirm B's record reports B's workspace.
	got, err := pgStore.Get(recB.Code)
	if err != nil {
		t.Fatalf("get B: %v", err)
	}
	if got.WorkspaceID != wsB {
		t.Fatalf("Get(%q) reported workspace %d, want %d", recB.Code, got.WorkspaceID, wsB)
	}

	// Update scoped to workspace A must not touch B's URL.
	newURL := "https://evil.example/hijack"
	if _, err := pgStore.Update(data.URLUpdateParams{
		Code: recB.Code, WorkspaceID: wsA, LongURL: &newURL,
	}); err == nil {
		t.Fatalf("cross-workspace update unexpectedly succeeded")
	}
	stillB, _ := pgStore.Get(recB.Code)
	if stillB.LongURL != "https://example.com/b" {
		t.Fatalf("cross-workspace update mutated B's URL to %q", stillB.LongURL)
	}

	// Delete scoped to workspace A must not delete B's URL.
	if err := pgStore.Delete(wsA, recB.Code); err == nil {
		t.Fatalf("cross-workspace delete unexpectedly succeeded")
	}
	if _, err := pgStore.Get(recB.Code); err != nil {
		t.Fatalf("B's URL was deleted by workspace A: %v", err)
	}

	// Sanity: owning workspace can still operate on its own URL.
	if _, err := pgStore.Update(data.URLUpdateParams{
		Code: recA.Code, WorkspaceID: wsA, LongURL: &newURL,
	}); err != nil {
		t.Fatalf("same-workspace update failed: %v", err)
	}
}

// TestRowLevelSecurity verifies the urls RLS policy hides other tenants' rows from
// a connection scoped to a single workspace (Phase 14.7). Skips when the DSN role
// is a superuser, which bypasses RLS.
func TestRowLevelSecurity(t *testing.T) {
	pgStore, authStore, pool := setup(t)
	defer pgStore.Pool.Close()

	var superuser bool
	if err := pool.QueryRow(context.Background(),
		`SELECT current_setting('is_superuser') = 'on'`).Scan(&superuser); err != nil {
		t.Fatalf("check superuser: %v", err)
	}
	if superuser {
		t.Skip("DSN role is a superuser; RLS is bypassed. Use the app's non-privileged role.")
	}

	userA := makeUser(t, authStore, "rls-a")
	userB := makeUser(t, authStore, "rls-b")
	wsA := *userA.DefaultWorkspaceID
	wsB := *userB.DefaultWorkspaceID

	recB, err := pgStore.Create(data.URLCreateParams{
		LongURL: "https://example.com/rls-b", RedirectType: 302, IsCrawlable: true,
		UserID: &userB.ID, WorkspaceID: wsB,
	})
	if err != nil {
		t.Fatalf("create url B: %v", err)
	}

	// A raw query scoped to workspace A must not see workspace B's row.
	tx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	if _, err := tx.Exec(context.Background(),
		`SELECT set_config('app.current_workspace_id', $1, true)`,
		strconv.FormatInt(wsA, 10)); err != nil {
		t.Fatalf("set guc: %v", err)
	}

	var count int
	if err := tx.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM urls WHERE code = $1`, recB.Code).Scan(&count); err != nil {
		t.Fatalf("scoped count: %v", err)
	}
	if count != 0 {
		t.Fatalf("RLS failed: workspace A saw %d rows of workspace B's URL", count)
	}
}
