# Coding Conventions

**Analysis Date:** 2026-08-31

This repo has two independently-versioned codebases sharing one git history:
- **Go backend** (`backend/`) — gRPC service + business logic, Go 1.25
- **Go frontend server** (`frontend/`) — thin REST gateway wrapper (grpc-gateway), Go 1.24
- **React/TypeScript UI** (`frontend/ui-src/`) — Vite + React 19 SPA, built and embedded into the frontend Go binary (`frontend/webapp`)

No `.golangci.yml` is committed, but CI runs `golangci-lint-action` with `version: latest` against both `backend/` and `frontend/` (`.github/workflows/*.yml`), so default golangci-lint rules apply — write lint-clean code (no unused vars/imports, `gofmt`-formatted, no shadowed errors).

No ESLint/Prettier config is committed for the UI. CI only runs `npx tsc --noEmit` for the `ui` job (`.github/workflows/ci.yml`), so **TypeScript's strict compiler is the only enforced style gate** for the frontend — see `frontend/ui-src/tsconfig.json` (`strict: true`, `noUnusedLocals: true`, `noUnusedParameters: true`, `noFallthroughCasesInSwitch: true`).

## Naming Patterns

**Go files:**
- One concern per file, named after the domain noun: `backend/auth/store.go`, `backend/auth/membership.go`, `backend/auth/invitation.go`, `backend/auth/jwt.go`, `backend/auth/oidc.go`, `backend/auth/password.go`, `backend/auth/workspace.go`
- gRPC service implementations live in the domain package next to the service they back: `backend/shortener/shortener.go` (URL CRUD), `backend/shortener/auth_service.go` (auth gRPC methods), `backend/shortener/analytics.go`, `backend/shortener/tags.go`, `backend/shortener/qr.go`
- Validation helpers are isolated: `backend/shortener/validate.go`
- Data-access implementations: `backend/data/postgres.go`, `backend/data/redis.go`, `backend/data/cached_store.go`, `backend/data/tags.go`, `backend/data/analytics.go`, `backend/data/visit.go`, `backend/data/useragent.go`, `backend/data/codegen.go`, `backend/data/migrate.go`
- REST gateway extras beyond grpc-gateway auto-routes: `backend/gateway/admin.go`, `backend/gateway/members.go`, `backend/gateway/workspace.go`, `backend/gateway/swagger_ui.go`

**Go identifiers:**
- Exported types are `PascalCase` nouns: `AuthStore`, `CreateServer`, `URLCreateParams`, `URLListParams`, `APIKey`, `SignInEvent`
- Constructors are `NewX`: `NewAuthStore(pool *pgxpool.Pool) *AuthStore`, `NewPostgresStore(...)`
- Store/service structs hold a `*pgxpool.Pool` (or embedded store) as their only field and expose methods as the public API — no interfaces are defined for `AuthStore` itself; interfaces (`data.URLStore`) exist only where multiple backends are swapped in (see `backend/data/store.go`)
- Unexported package-level sentinel errors are `camelCase` prefixed with `err`: `errNoURL`, `errInvalidURL`, `errSlugTooShort` (`backend/shortener/shortener.go:18-23`)
- Exported sentinel errors used across packages are `ErrX`: `auth.ErrLastOwner` (`backend/auth/membership.go:40`)
- Package-level loggers are declared once per package as `var log = hclog.Default()`: `backend/auth/store.go:16`, `backend/gateway/gateway.go:18`, `backend/shortener/shortener.go:34`
- SQL column lists are hoisted into package-level `const` strings for reuse across scan helpers: `const userColumns = ...` (`backend/auth/store.go:14`)
- Scan helpers are private functions named `scanX`: `scanUser(row interface{ Scan(...any) error }) (*User, error)` (`backend/auth/store.go:16`)

**TypeScript/React files:**
- Page components: `PascalCase.tsx` under `frontend/ui-src/src/pages/` (`CreateURL.tsx`, `EditURL.tsx`, `Dashboard.tsx`), with an `admin/` subfolder for admin-only screens (`frontend/ui-src/src/pages/admin/Users.tsx`, `frontend/ui-src/src/pages/admin/OIDCProviders.tsx`)
- Shared components: `PascalCase.tsx` under `frontend/ui-src/src/components/` (`TagInput.tsx`, `Layout.tsx`, `WorkspaceSwitcher.tsx`)
- Hooks: `camelCase.ts` prefixed `use` under `frontend/ui-src/src/hooks/` (`useAuth.ts`)
- API client: single file `frontend/ui-src/src/api/client.ts` exporting grouped namespace objects (`auth`, `admin`, `urls`, `tags`, ...), not one file per resource
- Shared types: `frontend/ui-src/src/types/index.ts`

**React component/props naming:**
- Default-export function components named identically to the file: `export default function CreateURL()`, `export default function TagInput(...)`
- Props interfaces are named `<Component>Props` and declared directly above the component: `interface TagInputProps { value: string[]; onChange: (tags: string[]) => void }` then `export default function TagInput({ value, onChange }: TagInputProps)` (`frontend/ui-src/src/components/TagInput.tsx:5-9`)
- Local component state uses `useState` with paired `camelCase` getter and `setX` setter: `const [loading, setLoading] = useState(false)`

## Code Style

**Go formatting:**
- Standard `gofmt`; multi-return-value function signatures wrap params across lines when long (`backend/auth/store.go:47`)
- Doc comments precede every exported type/function, starting with the identifier name: `// CreateUser inserts a new user and auto-provisions a personal workspace...` (`backend/auth/store.go:46`)
- Inline comments reference the phase/ticket that introduced the behavior, e.g. `// Phase 14`, `// (Phase 15.1)` — this is a strong convention throughout `backend/auth/*.go` and `backend/integration/*_test.go`; new code touching multi-tenancy/membership logic should follow suit and cite the phase or reason for non-obvious business rules
- Multi-line SQL literals are inline backtick strings inside the calling method, not extracted to separate files (`backend/auth/store.go:63-68`)

**TypeScript formatting:**
- No semicolons at statement ends (Standard/Prettier-less style observed throughout `frontend/ui-src/src/**/*.ts(x)`)
- Single quotes for strings: `'/api/v1'`, `'Content-Type'`
- 2-space indentation
- Arrow functions preferred for handlers and API client methods; `function` keyword reserved for component definitions (`export default function CreateURL()`)

## Import Organization

**Go import order** (stdlib, blank line, third-party, blank line, internal — goimports-style), e.g. `backend/auth/store.go:3-13`:
```go
import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)
```
Internal packages use the full module path, no aliasing except for generated protobuf code, which is conventionally aliased `pb`:
```go
pb "github.com/incidrthreat/goshorten/backend/pb"
```
(`backend/shortener/shortener.go:8`, `backend/gateway/gateway.go:13`)

**TypeScript import order:** React/external libs first, then relative imports, e.g. `frontend/ui-src/src/pages/CreateURL.tsx:1-5`:
```ts
import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { urls } from '../api/client'
import { Copy, Check } from 'lucide-react'
import TagInput from '../components/TagInput'
```
No path aliases configured — all cross-directory imports use relative paths (`../api/client`, `../components/...`).

## Error Handling

**Go — repository/domain layer:** wrap every error with `fmt.Errorf("<operation context>: %w", err)` so call chains stay debuggable:
```go
u, err := scanUser(s.Pool.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE email = $1`, email))
if err != nil {
	return nil, fmt.Errorf("get user by email: %w", err)
}
```
(`backend/auth/store.go:105-109`)

Not-found is distinguished from real errors at the lowest layer by translating `pgx.ErrNoRows` into a `nil, nil` return, letting callers treat "not found" as a normal, non-error case:
```go
if err != nil {
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return nil, err
}
```
(`backend/auth/store.go:20-24`)

Sentinel errors (`errors.New(...)`) are declared package-level for conditions callers need to compare against with `errors.Is`, e.g. `auth.ErrLastOwner` (`backend/auth/membership.go:40`), asserted in tests via `errors.Is(err, auth.ErrLastOwner)` (`backend/integration/membership_test.go:37`).

Ad-hoc user-facing validation errors are created inline with `errors.New("...")` rather than sentinels when the message is unique to one call site: `errors.New("email is required")`, `errors.New("that user is already a member of this workspace")` (`backend/auth/invitation.go:60,79`).

**Go — gRPC service layer:** every handler translates internal errors into a `status.Error(codes.X, "message")` before returning to the client — never leak raw internal errors across the gRPC boundary:
```go
if err != nil {
	return nil, status.Error(codes.Internal, "authentication failed")
}
if !ok {
	return nil, status.Error(codes.Unauthenticated, "invalid credentials")
}
```
(`backend/shortener/auth_service.go:35-46`)
Common codes used: `codes.NotFound`, `codes.PermissionDenied`, `codes.InvalidArgument`, `codes.Unauthenticated`, `codes.FailedPrecondition`, `codes.Internal`.

**Go — transactions:** always `defer func() { _ = tx.Rollback(ctx) }()` immediately after `Begin`, then commit explicitly at the end; rollback errors are intentionally discarded since a successful commit makes rollback a no-op (`backend/auth/store.go:52-53`, `backend/auth/invitation.go:170+`).

**TypeScript — API client:** all HTTP calls funnel through a single `request<T>()` wrapper in `frontend/ui-src/src/api/client.ts` that throws a typed `APIError` (extends `Error`, carries `status`) on non-2xx responses, attempting to parse a JSON `{message}`/`{error}` body first:
```ts
class APIError extends Error {
  constructor(public status: number, message: string) {
    super(message)
  }
}
```
(`frontend/ui-src/src/api/client.ts:3-7`)

**TypeScript — component layer:** `try/catch` around API calls, narrowing `err` with `err instanceof Error` before using `.message`, falling back to a generic string:
```ts
} catch (err) {
  setError(err instanceof Error ? err.message : 'Create failed')
} finally {
  setLoading(false)
}
```
(`frontend/ui-src/src/pages/CreateURL.tsx:41-45`)
Non-fatal background fetches (e.g. hydrating theme/account after login) use empty or comment-only `catch` blocks: `catch { // non-fatal }` (`frontend/ui-src/src/hooks/useAuth.ts:49-51`).

## Logging

**Framework:** `github.com/hashicorp/go-hclog`, one package-level `var log = hclog.Default()` per Go package that logs (`backend/auth/store.go:16`, `backend/gateway/gateway.go:18`, `backend/shortener/shortener.go:34`).

**Frontend:** no logging framework; browser `console` is not used in the reviewed UI code — errors surface via component state (`setError`) instead.

## Comments

**Go:** every exported symbol has a doc comment starting with its name (standard Go doc convention), e.g. `// AuthStore handles auth-related database operations.` (`backend/auth/store.go:79`). Non-obvious business rules get an inline comment citing the phase that introduced them, e.g. `// The ON CONFLICT guard tolerates re-runs / legacy migration backfills.` (`backend/auth/store.go:70-71`).

**TypeScript:** sparse comments; used mainly to explain non-obvious multi-tenancy/session behavior, e.g. `// Active workspace scope (Phase 14). Sent on every request so both the grpc-gateway and the self-service endpoints resolve the same tenant.` (`frontend/ui-src/src/api/client.ts:17-18`).

## Function Design

**Go:** methods on store/server structs take a `context.Context` first, then explicit scalar params or a `XxxParams` struct for anything with more than ~3 fields (`URLCreateParams`, `URLListParams`, `URLUpdateParams` in `backend/data/store.go`). Multi-tenant methods take an explicit `workspaceID int64` parameter rather than reading it from a struct field, to keep tenant scoping visible at every call site.

**TypeScript:** API client methods are one-line arrow functions returning `request<ResponseType>(path, options)`; response shape is expressed as an inline generic type argument rather than a named interface for simple/one-off shapes, and against `types/index.ts` for shared domain types.

## Module Design

**Go exports:** packages export structs with methods (`AuthStore`, `CreateServer`, `PostgresStore`) rather than free functions; there is no barrel/re-export file — consumers import the concrete package (`backend/auth`, `backend/data`, `backend/shortener`) directly.

**TypeScript exports:** `frontend/ui-src/src/api/client.ts` exports one const object per resource domain (`export const auth = {...}`, `export const admin = {...}`) acting as a lightweight namespace/barrel for that resource's calls — this is the pattern to follow when adding a new API resource (add a new exported const object, not a new file, unless the resource grows large).

---

*Convention analysis: 2026-08-31*
