# Quant Platform — Milestone 1: User Account Service with Dynamic RBAC (`apps/account`)

- **Date:** 2026-08-31
- **Status:** Draft for review
- **Scope:** New standalone Go service `apps/account` for user-facing identity, dynamic RBAC, watchlists, and admin. `apps/api` remains an internal-only market-data proxy and is NOT modified in M1.
- **Decisions locked:** golang-migrate (migrations); Redis (refresh tokens + permission cache); flat per-user watchlist; new single service `apps/account`; `apps/api` is internal-only; **dynamic RBAC** (roles/permissions assignable at runtime).
- **Guiding skills:** golang-security, golang-database, golang-dependency-injection, golang-testing.

---

## 1. Goal

Stand up `apps/account` — a Clean-Architecture Go service that owns:

- **Authentication**: register, login, refresh, logout (email + password; bcrypt; JWT).
- **Dynamic RBAC**: roles and permissions are assignable at runtime via admin endpoints. Authorization is checked by **permission**, not by a static role string.
- **Users**: self profile read/update; admin management of users.
- **Watchlists**: per-user symbol watchlists (flat list for M1).
- **Admin**: role-gated by permission, with full role/permission/assignment management.

`apps/api` is unchanged and stays an internal market proxy. No `api ↔ account` HTTP coupling in M1.

## 2. Non-goals (M1)

- No market-data analysis, charts, screening, or backtesting (M3–M4).
- No `apps/api` changes; `api` stays internal-only.
- No OAuth/SSO, email verification, or password-reset flows (can follow later).
- No inter-service auth delegation; tokens are issued and verified entirely within `apps/account`.

## 3. Architecture

`apps/account` follows the **exact** conventions of `apps/api`:

- Layer order: `cmd/server/main.go` → `internal/container` (**samber/do v2**, composition root — the only place DI happens) → `delivery/http` → `usecase` → `repository` (contracts) → `infrastructure` (database, cache, config).
- Router: `chi/v5` + `slog-chi`, with `middleware.Recoverer`, `RequestID`, timeout, and rate limit (copy `apps/api/internal/delivery/http/middleware/ratelimit.go`).
- Config: `viper` loaded from `config.account.yaml` (mirror `config.api.yaml` shape).
- Database: `jackc/pgx/v5` **pgxpool** (no ORM; explicit SQL, parameterized). Reuse the `Postgres` wrapper + `TxManager` pattern from `apps/api`.
- Cache: `go-redis/v9` via a `Redis` wrapper (mirror `apps/api/internal/infrastructure/cache/redis.go`).
- DI: `samber/do/v2` used only in `container.go`; define repository/usecase interfaces at consumption sites; inject via constructors.
- Testing: `testify`, `pgxmock`, `miniredis`, `uber-go/mock` (typed mocks via `//go:generate mockgen`), table-driven with named subtests.

### 3.1 Directory layout

```text
apps/account/
  cmd/server/main.go
  go.mod                         # module github.com/nofendian17/sbterm/apps/account
  Dockerfile
  config.account.yaml.example
  internal/
    container/container.go
    delivery/http/
      router.go
      server.go
      middleware/
        ratelimit.go             # copied from apps/api
        auth.go                  # JWT validation + context identity (user_id, permission set)
      auth/ user/ watchlist/ admin/   # handler packages
    usecase/
      auth.go user.go watchlist.go rbac.go admin.go
    repository/
      auth.go user.go watchlist.go rbac.go   # contracts
    domain/
      user.go watchlist.go rbac.go errors.go
    infrastructure/
      config/config.go
      database/postgres.go transaction.go
      cache/redis.go
      repository/                # pgx + redis implementations
  migrations/
    account/
      000001_create_users.up.sql / .down.sql
      000002_create_rbac.up.sql / .down.sql       # roles, permissions, role_permissions, user_roles + seed
      000003_create_watchlists.up.sql / .down.sql
```

### 3.2 `go.work`

Add `./apps/account` to the existing `go.work` `use (...)` block.

## 4. Data model (Postgres, golang-migrate)

Schema is versioned, reviewed, and applied via golang-migrate (embedded `*.sql` via `iofs`). No triggers, views, or stored procedures (DB skill: keep SQL explicit in Go). All queries parameterized with pgx `$1` placeholders.

### `users`

```sql
CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    display_name  TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ
);
CREATE INDEX idx_users_email ON users (email);
CREATE INDEX idx_users_deleted_at ON users (deleted_at);
```
Role is **not** a column — it is relational via `user_roles`. Soft-delete via `deleted_at IS NULL`.

### RBAC

```sql
CREATE TABLE roles (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE permissions (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resource   TEXT NOT NULL,
    action     TEXT NOT NULL,
    name       TEXT NOT NULL UNIQUE,   -- canonical key: "<resource>:<action>"
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE role_permissions (
    role_id       UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE user_roles (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);

CREATE INDEX idx_user_roles_user ON user_roles (user_id);
CREATE INDEX idx_role_permissions_role ON role_permissions (role_id);
```

Seed (in `000002` up): permissions (`auth:login`, `profile:read`, `profile:write`, `watchlist:read`, `watchlist:write`, `admin:roles:read`, `admin:roles:write`, `admin:users:read`, `admin:users:manage`, `admin:rbac:assign`); role `user` (granted the non-admin permissions); role `admin` (granted all). New registrations are assigned role `user` inside the register transaction.

### `watchlists`

```sql
CREATE TABLE watchlists (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    symbol     TEXT NOT NULL,
    label      TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, symbol)
);
CREATE INDEX idx_watchlists_user_id ON watchlists (user_id);
```
`symbol` is free text in the Stockbit symbol namespace (no FK to market data — preserves separation; `apps/api` owns market data).

## 5. Authentication & authorization design (security skill applied)

- **Password hashing**: `golang.org/x/crypto/bcrypt` at cost **12**. Verify with `bcrypt.CompareHashAndPassword` (constant-time internally). Never compare hashes with `==`.
- **JWT**: `github.com/golang-jwt/jwt/v5`. Claims: `sub`=user id, `type`(`access`|`refresh`), `jti`, `exp`, `iat`. Signed with `auth.jwt_secret` from config.
  - **Secret hygiene**: `auth.jwt_secret` MUST be non-empty; fail fast at startup if empty in non-`dev` mode. Supplied via `config.account.yaml` (mounted read-only from a secret store in prod) — never hardcoded.
- **Access token**: short-lived (`auth.access_ttl`, default 15m), stateless, verified by `AuthMiddleware`.
- **Refresh token**: opaque random id generated with `crypto/rand` (NOT `math/rand`), stored in **Redis** under `refresh:<jti>` with TTL = `auth.refresh_ttl`. Logout/rotation deletes the key. Rotation issues a new `jti` and deletes the old.
- **Sessions invalidation**: a per-user `token_version` counter (Redis or `users` column). Admin suspend/role change bumps it; `AuthMiddleware` rejects access tokens issued before the bump. (Defense-in-depth beyond refresh-key deletion.)
- **Rate limiting**: `rate_limit` applied globally; auth endpoints additionally protected from brute force (tight burst).
- **Generic errors**: never return DB/stack details to clients; log server-side with `slog`.
- **Endpoints** (under `/api/v1`):
  - `POST /api/v1/auth/register` — public. Validates (go-playground/validator) email/password; hashes; inserts user + assigns `user` role in **one transaction**; issues tokens.
  - `POST /api/v1/auth/login` — public. Verifies bcrypt; issues tokens; stores refresh in Redis.
  - `POST /api/v1/auth/refresh` — requires refresh token; rotates.
  - `POST /api/v1/auth/logout` — authenticated; deletes refresh key + bumps token_version.
- **Open routes**: `/healthz`, `/api/v1/auth/register`, `/api/v1/auth/login`.

### 5.1 `AuthMiddleware`

- Reads `Authorization: Bearer <access>`, verifies signature + `type==access` + not-before token_version.
- Injects `user_id` and the resolved **permission set** into `context.Context` (typed keys).
- 401 on missing/invalid/expired; 403 on missing permission (enforced per-route via `RequirePermission(...)` middleware or in usecase).

### 5.2 Permission resolution & caching

- `rbac` usecase/repository resolves a user's permissions via `user_roles → role_permissions`.
- Cache the resolved set in **Redis** under `perms:<user_id>` (TTL ~5m), invalidated on any role/permission assignment change. Fallback to DB lookup on miss.

## 6. Domain APIs

### Users (`/api/v1/users`) — permission `profile:read` / `profile:write`
- `GET /api/v1/users/me` — own profile.
- `PUT /api/v1/users/me` — update `display_name` / change password.

### Watchlists (`/api/v1/watchlists`) — `watchlist:read` / `watchlist:write`
- `GET` list own; `POST` add `{symbol, label?}`; `DELETE /{symbol}` remove.

### Admin RBAC (`/api/v1/admin`) — admin permissions
- `GET/POST /api/v1/admin/roles` — list/create roles (`admin:roles:read`/`write`).
- `GET/PUT/DELETE /api/v1/admin/roles/{id}` — manage a role.
- `POST/DELETE /api/v1/admin/roles/{id}/permissions` — assign/unassign permission to role (`admin:rbac:assign`).
- `GET/POST/DELETE /api/v1/admin/users/{id}/roles` — assign/unassign role to user (`admin:rbac:assign`).
- `GET /api/v1/admin/users` — list users (paginated, respects soft-delete) (`admin:users:read`).
- `GET /api/v1/admin/users/{id}` — view one (`admin:users:read`).
- `POST /api/v1/admin/users/{id}/suspend` — soft-delete/suspend (`admin:users:manage`).
- `DELETE /api/v1/admin/users/{id}` — soft delete (`admin:users:manage`).
- `GET /api/v1/admin/users/{id}/watchlists` — view user's watchlists (`admin:users:read`).

> Authorization authority is the usecase (`HasPermission(ctx, perm)`), not just middleware, so logic is testable and centralized.

## 7. Configuration (`config.account.yaml`)

```yaml
app:
  name: account
  version: dev
port: ":8081"
database:
  url: postgres://postgres:postgres@postgres:5432/sbterm?sslmode=disable
  max_conns: 10
  min_conns: 0
  max_conn_lifetime: 1h
  max_conn_idle_time: 30m
redis:
  url: redis://redis:6379/0
  max_retries: 3
  pool_size: 10
  min_idle_conns: 0
  dial_timeout: 5s
  read_timeout: 3s
  write_timeout: 3s
log:
  level: info
  format: json
  add_source: false
rate_limit:
  rate: 50
  burst: 100
auth:
  jwt_secret: change-me-in-prod
  access_ttl: 15m
  refresh_ttl: 720h
  bcrypt_cost: 12
http:
  read_timeout: 10s
  write_timeout: 10s
  idle_timeout: 60s
```

Provide `config.account.yaml.example`. Port `:8081` to avoid `apps/api` at `:8080`. **`jwt_secret` must be empty-by-default in example and required in prod.**

## 8. Deployment

- `apps/account/Dockerfile` (multi-stage; build arg `APP_VERSION` ldflags into `internal/infrastructure/config.version`).
- `account` service in `docker-compose.yml` (depends_on `postgres` + `redis` healthy; mounts `config.account.yaml`; publishes host port `8081`).
- `GET /healthz` → 200 for compose healthcheck.

## 9. Testing (golang-testing + database + security applied)

- **Table-driven** with named subtests for every handler/usecase/repo; `assert`/`require` instances built per-subtest (never bound to parent `t`).
- **Unit — domain/usecase**: bcrypt verify, JWT round-trip (sign/verify with wrong secret fails), register duplicate-email (unique-constraint → conflict error), refresh rotation, **permission gating** (user without `admin:users:read` is denied), role assignment changes effective permissions. Mock repositories with `uber-go/mock` typed mocks.
- **Repository (pgx)** with `pgxmock`: user insert/lookup, soft-delete filter (`deleted_at IS NULL`), unique-constraint detection; RBAC joins (`user_roles→role_permissions`); watchlist upsert/list/delete. Always `QueryContext`/`ExecContext` with `$N` params; `defer rows.Close()`; translate `sql.ErrNoRows` → domain `ErrNotFound`.
- **Repository (Redis)** with `miniredis`: refresh store/delete/rotate; permission-set cache write/invalidate.
- **HTTP** with `httptest`: table tests per handler including `AuthMiddleware` 401/403/role cases; request validation errors return 400 with generic messages.
- **Integration** (`//go:build integration`): migrations apply, register→login→authed-call flow against a real Postgres (testcontainer/CI). Run separately: `go test -tags=integration ./...`.
- **Race**: `go test -race ./...` in CI; `goleak.VerifyTestMain` where goroutines are spawned.
- **Migrations**: SQL authored/reviewed as code (golang-migrate), not hand-rolled at runtime.

## 10. Roadmap (post-M1)

- **M2** Price alerts (reuse Stockbit `notification` concept) + portfolio basics.
- **M3** Technical charts/indicators over existing `chartbit` proxy.
- **M4** Screening + backtesting — requires reading local QuestDB (the one genuinely new capability); likely a new `apps/api` read path or a dedicated analytics service.

## 11. Decisions resolved

- Service layout: **1 new service `apps/account`**.
- `apps/api`: **internal-only**, unchanged in M1.
- Auth: bcrypt (cost 12) + `golang-jwt/jwt/v5`; access stateless; refresh in **Redis** with `crypto/rand` jti; `token_version` invalidation.
- Authorization: **dynamic RBAC** by permission (roles/permissions/assignments managed at runtime via admin endpoints), cached in Redis.
- Migrations: **golang-migrate** (embedded).
- Watchlist: flat per-user.
- DB access: **pgx, no ORM**, parameterized; `samber/do` DI exactly like `apps/api`.

## 12. Self-review

- No `TBD`/placeholder sections.
- Consistent: RBAC tables match repo/usecase; admin endpoints gated by specific permissions; register transaction assigns default role.
- Scope: single new service — fits one implementation plan.
- Ambiguity resolved: "suspend" = soft-delete for M1; refresh token = opaque `crypto/rand` id in Redis; `symbol` free text; secrets via config, required in prod.
