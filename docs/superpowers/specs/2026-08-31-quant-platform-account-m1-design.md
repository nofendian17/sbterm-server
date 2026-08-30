# Quant Platform — Milestone 1: User Account Service (`apps/account`)

- **Date:** 2026-08-31
- **Status:** Draft for review
- **Scope:** New standalone Go service `apps/account` for user-facing identity and state. `apps/api` remains an internal-only market-data proxy and is NOT modified in M1.
- **Decisions locked:** golang-migrate (migrations); Redis (refresh tokens); flat per-user watchlist; new single service `apps/account`; `apps/api` is internal-only.

---

## 1. Goal

Stand up `apps/account` — a Clean-Architecture Go service that owns:

- **Authentication**: register, login, refresh, logout (email + password, bcrypt, JWT).
- **Users**: self profile read/update; admin management of users.
- **Watchlists**: per-user symbol watchlists (flat list for M1).
- **Admin**: role-gated user management.

`apps/api` is unchanged and stays an internal market proxy. No `api ↔ account` HTTP coupling in M1 — the client talks to `account` for user concerns and to `api` (internally) for market data. Inter-service calls are explicitly **out of scope for M1**.

## 2. Non-goals (M1)

- No market-data analysis, charts, screening, or backtesting (M3–M4).
- No `apps/api` changes; `api` stays internal-only.
- No OAuth/SSO, email verification, or password-reset flows (can follow later).
- No inter-service auth delegation; tokens are issued and verified entirely within `apps/account`.

## 3. Architecture

`apps/account` follows the **exact** conventions established by `apps/api`:

- Layer order: `cmd/server/main.go` → `internal/container` (samber/do v2, composition root) → `delivery/http` → `usecase` → `repository` (contracts) → `infrastructure` (database, cache, config).
- Router: `chi/v5` + `slog-chi`, with `middleware.Recoverer`, `RequestID`, timeout, and rate limit (reuse the same pattern as `apps/api/internal/delivery/http/middleware/ratelimit.go`).
- Config: `viper` loaded from `config.account.yaml` (mirror `config.api.yaml` shape).
- Database: `jackc/pgx/v5` `pgxpool` via a `Postgres` wrapper + `TxManager` (mirror `apps/api/internal/infrastructure/database/postgres.go` and `transaction.go`).
- Cache: `go-redis/v9` via a `Redis` wrapper (mirror `apps/api/internal/infrastructure/cache/redis.go`).
- DI: `samber/do/v2` used **only** in `container.go`.
- Testing: `testify`, `pgxmock`, `miniredis`, `uber-go/mock` (mockgen with `//go:generate`, typed mocks) — same as `apps/api`.

### 3.1 Directory layout

```text
apps/account/
  cmd/server/main.go
  go.mod                       # module github.com/nofendian17/sbterm/apps/account
  Dockerfile
  internal/
    container/container.go
    delivery/http/
      router.go
      server.go
      middleware/
        ratelimit.go           # copied from apps/api
        auth.go                # JWT validation + context identity
      auth/                    # register/login/refresh/logout handlers
      user/                    # self profile handlers
      watchlist/               # CRUD handlers
      admin/                   # admin handlers
    usecase/
      auth.go  user.go  watchlist.go  admin.go
    repository/
      auth.go  user.go  watchlist.go  admin.go
    domain/
      user.go  watchlist.go  auth.go  errors.go
    infrastructure/
      config/config.go
      database/postgres.go  transaction.go
      cache/redis.go
      repository/             # postgres + redis implementations
  migrations/
    account/
      000001_create_users.up.sql
      000001_create_users.down.sql
      000002_create_watchlists.up.sql
      000002_create_watchlists.down.sql
```

### 3.2 Registration in `go.work`

Add `./apps/account` to the existing `go.work` `use (...)` block.

## 4. Data model (Postgres)

### `users`

```sql
CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    display_name  TEXT NOT NULL,
    role          TEXT NOT NULL DEFAULT 'user' CHECK (role IN ('user','admin')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ
);
CREATE INDEX idx_users_email ON users (email);
CREATE INDEX idx_users_deleted_at ON users (deleted_at);
```

Soft-delete via `deleted_at IS NULL` filter in queries. `role` enum enforced at DB level; default `user`.

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

Flat per-user symbol list (no nested folders in M1). `symbol` is the Stockbit symbol namespace already used by `apps/api` (no FK to market data — maintains separation).

### Migrations

- Tool: **golang-migrate**. Migrations embedded via `github.com/golang-migrate/migrate/v4/database/postgres` + `iofs` source (`embed` the `migrations/account` dir).
- Run at startup in `cmd/server/main.go` before the HTTP server starts (mirrors nothing existing, but standard). On failure, fail fast.

## 5. Authentication design

- **Password hashing**: `golang.org/x/crypto/bcrypt` (already an indirect dep in `apps/api`; add directly here).
- **JWT**: `github.com/golang-jwt/jwt/v5`. Claims: `sub` = user id, `role`, `type` (`access`|`refresh`), `exp`, `iat`. Signed with `auth.jwt_secret` from config.
- **Access token**: short-lived (config `auth.access_ttl`, default 15m), stateless, verified by `AuthMiddleware`.
- **Refresh token**: long-lived (config `auth.refresh_ttl`, default 30d), stored in **Redis** under key `refresh:<token_id>` (random id embedded in claims or token jti). Logout/refresh rotation deletes the old key; a `token_version` per user (stored on `users` or in Redis) lets admin/suspend invalidate all sessions at once.
- **Endpoints** (all under `/api/v1`):
  - `POST /api/v1/auth/register` — public. Validates email/password (go-playground/validator), hashes, inserts user (`role='user'`), issues tokens.
  - `POST /api/v1/auth/login` — public. Verifies bcrypt, issues tokens, stores refresh in Redis.
  - `POST /api/v1/auth/refresh` — requires refresh token (in body or `Authorization: Bearer`). Rotates refresh, issues new access.
  - `POST /api/v1/auth/logout` — authenticated. Deletes refresh key from Redis.
- **Open routes**: `/healthz`, `/api/v1/auth/register`, `/api/v1/auth/login`.

### 5.1 `AuthMiddleware`

- Reads `Authorization: Bearer <access>`, parses/verifies JWT with `auth.jwt_secret`, checks `type == access`.
- Injects `user_id` and `role` into `context.Context` (typed keys) for downstream handlers.
- 401 on missing/invalid/expired. Admin-gated handlers additionally check `role == admin` (or delegate to usecase).

## 6. Domain APIs

### Users (`/api/v1/users`)
- `GET /api/v1/users/me` — authenticated: own profile.
- `PUT /api/v1/users/me` — authenticated: update `display_name` (and password change if supplied + verified).

### Watchlists (`/api/v1/watchlists`)
- `GET /api/v1/watchlists` — authenticated: list own symbols.
- `POST /api/v1/watchlists` — authenticated: add `{symbol, label?}` (unique per user).
- `DELETE /api/v1/watchlists/{symbol}` — authenticated: remove.

### Admin (`/api/v1/admin`)
- `GET /api/v1/admin/users` — admin: list users (paginated, respects soft-delete).
- `GET /api/v1/admin/users/{id}` — admin: view one user.
- `POST /api/v1/admin/users/{id}/suspend` — admin: soft-delete / set inactive (sets `deleted_at` or a status). For M1, suspend = soft-delete; can be refined.
- `DELETE /api/v1/admin/users/{id}` — admin: hard or soft delete (choose soft for safety).
- `GET /api/v1/admin/users/{id}/watchlists` — admin: view a user's watchlists.

> Admin role check: `role` claim from JWT + enforced in usecase; middleware can pre-check but usecase is the authority.

## 7. Configuration (`config.account.yaml`)

Mirror `config.api.yaml`:

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
http:
  read_timeout: 10s
  write_timeout: 10s
  idle_timeout: 60s
```

Provide `config.account.yaml.example` (mirror existing `.example` convention). Port `:8081` to avoid clashing with `apps/api` at `:8080`.

## 8. Deployment

- Add `apps/account/Dockerfile` (multi-stage, copy `apps/api/Dockerfile` pattern; build arg `APP_VERSION` ldflags into `internal/infrastructure/config.version`).
- Add `account` service to `docker-compose.yml` (depends_on `postgres` + `redis` healthy; mounts `config.account.yaml`; publishes host port e.g. `8081`).
- Health endpoint `GET /healthz` returning 200; used by compose healthcheck.

## 9. Testing (TDD, per repo conventions)

- `domain`: value types, validator tags.
- `usecase`: bcrypt verify, JWT issue/verify round-trip, register duplicate-email error, refresh rotation, admin gating — with `uber-go/mock` typed mocks for repositories.
- `infrastructure/repository` (Postgres): `pgxmock` for user insert/lookup, watchlist upsert/list/delete, soft-delete filter; unique-constraint handling.
- `infrastructure/repository` (Redis): `miniredis` for refresh store/delete/rotate.
- `delivery/http`: `httptest` table tests per handler, including `AuthMiddleware` 401/role cases.
- `migration`: sanity that embedded SQL applies (optional integration test against testcontainer/real pg in CI — if not available, manual).

## 10. Roadmap (post-M1)

- **M2** Price alerts (reuse Stockbit `notification` domain concept) + portfolio basics.
- **M3** Technical charts/indicators over existing `chartbit` proxy (in `apps/api` or proxied internally).
- **M4** Screening + backtesting — will require reading local QuestDB (the one genuinely new capability); likely a new read path in `apps/api` or a dedicated analytics service.

## 11. Open questions resolved

- Service layout: **1 new service `apps/account`** (Option 2 / single-service simplification chosen by user).
- `apps/api`: **internal-only**, unchanged in M1.
- Migrations: **golang-migrate**.
- Refresh tokens: **Redis**.
- Watchlist shape: **flat per-user**.

## 12. Self-review notes

- No `TBD`/placeholder sections.
- Internal consistency: architecture mirrors `apps/api`; DB schema matches domain/repo; admin role enforced in usecase.
- Scope: M1 is a single new service — fits one implementation plan.
- Ambiguity resolved: "suspend" = soft-delete for M1; refresh token stored by `jti` in Redis; watchlist `symbol` is free text in Stockbit namespace.
