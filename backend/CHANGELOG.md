# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## [Unreleased]
- Versioning policy update: phase changes are tracked as **minor** releases in `0.x.0` format (for example, `0.5.0`), not major version bumps.

### Phase 15: Memberships, Roles & Invitations
#### Added
- `memberships` table (user_id, workspace_id, role, status) and `invitations` table (workspace_id, email, role, token, invited_by, expires_at, status) with migration 000009. Existing workspace owners and legacy shared-workspace users are backfilled into memberships.
- Per-workspace roles — Owner, Admin, Member, Viewer — resolved by `AuthStore.WorkspaceRole`. The gRPC interceptor enforces viewer read-only access and exposes the role via context (`WorkspaceRoleFromContext`).
- "At least one Owner" invariant: the last owner cannot be demoted or removed (`ErrLastOwner`), plus an ownership-transfer flow (`TransferOwnership`).
- Invitation lifecycle: create → email accept link → membership on accept, with resend, revoke, and an hourly expiry-sweep worker. Accept handles both existing accounts (sign in) and new invitees (self-service signup for the invited email).
- Member/invitation REST endpoints under `/api/v1/workspaces/{id}/members`, `/transfer-ownership`, `/invitations`, and token-addressed `/api/v1/invitations/{token}` (public preview + accept).
- `mail` package: SMTP sender (STARTTLS / implicit TLS) with a log-only fallback so invitations work without an SMTP server; create/resend responses also return a copyable accept link. Configurable via `GOSHORTEN_SMTP_*`, `GOSHORTEN_SMTP_FROM`, and `GOSHORTEN_APP_BASE_URL`.
- Membership isolation, last-owner, ownership-transfer, and invite-accept integration tests (`backend/integration`).
#### Changed
- `AuthStore.CreateUser` now also creates an owner membership for the auto-provisioned personal workspace; `ListWorkspacesForUser`/`UserCanAccessWorkspace` resolve access via memberships (with owner/default-workspace bridges for legacy accounts).
- Global `users.role` remains the platform/operator flag; day-to-day authorization within a workspace is governed by the membership role.

### Phase 14: Multi-Tenancy Foundation (Workspaces & Tenant Isolation)
#### Added
- `workspaces` table (id, name, slug, owner_id, plan, created_at) as the billable tenant unit, plus `workspace_id` foreign keys on `urls`, `clicks`, `tags`, `api_keys`, and `domains` (migration 000007). Existing rows are backfilled into a single "Default" workspace owned by the earliest admin.
- Auto-provisioning of a personal workspace on user creation (`AuthStore.CreateUser`), so no account is ever workspace-less. Users may also create additional workspaces they own.
- Active-workspace resolution in the gRPC auth interceptor: `X-Workspace-Id` header override → session's last-switched workspace → user default. API keys are bound to a single workspace.
- Workspace self-service REST endpoints (`/api/v1/workspaces`, `/workspaces/switch`, `/workspaces/{id}`) and a dashboard workspace switcher that persists the active workspace per session.
- Row-Level Security backstop on `urls` (migration 000008) keyed on `app.current_workspace_id`, enforced via per-request transaction helpers in the repository layer.
- Cross-tenant isolation integration tests (`backend/integration`, gated on `GOSHORTEN_TEST_DSN`).
#### Changed
- Every workspace-scoped repository query (`Create`/`Get`/`Update`/`Delete`/`List`, tags, analytics access) now enforces `workspace_id`; tag names are unique per workspace rather than globally.
- Platform/operator admin (global `role=admin`) is distinct from workspace ownership: operator URL search runs unscoped while normal URL/tag/analytics operations are confined to the active workspace.

## [0.0.5] - 2020-08-19
### Added 
- gRPC server GetStats message to retrieve Statistics from Redis 
### Edited
- CreateURL now stores the initial stats data

## [0.0.4] - 2020-08-10
### Added
- Converted TTL from string to int64
- Added server/client Keepalive

## [0.0.3] - 2020-08-09
### Added
- Added grpc Time-To-Live save and get func

## [0.0.2] - 2020-07-25
### Added
- backend/main.go
  - added TLS secure connections from grpc client (frontend-go)
### Removed
- TLS pub/priv keys

## [0.0.1a] - 2020-07-15
### Edited
- backend/data/redis.go
  - edited generate() function to detect collisions with set limits on length of code and # of code generation attempts.  Function will return a warning/error when limits are hit.

## [0.0.1] - 2020-07-14
### Added
- Initial build