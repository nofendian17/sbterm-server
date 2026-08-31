# Quant Platform — `apps/core` M1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build `apps/core`, a new standalone Go service (Clean Architecture, samber/do DI) providing email+password auth, dynamic RBAC, per-user watchlists, and admin management, while `apps/api` stays internal-only and unchanged.

**Architecture:** `cmd/server/main.go` → `internal/container` (samber/do v2, the only DI site) → `delivery/http` (chi + slog-chi) → `usecase` → `repository` (contracts) → `infrastructure` (pgxpool, go-redis, viper, golang-migrate). One new Go module + docker-compose entry + embedded SQL migrations. All DB access via `pgx` `Querier`/`TxManager` (no ORM), fully parameterized.

**Tech Stack:** Go 1.26.5, chi/v5, samber/do/v2, jackc/pgx/v5 (pgxpool), go-redis/v9, golang-jwt/jwt/v5, golang.org/x/crypto/bcrypt, golang-migrate, viper, go-playground/validator, slog, testify, pgxmock, miniredis, uber-go/mock (mockgen -typed).

**Spec:** `docs/superpowers/specs/2026-08-31-quant-platform-account-m1-design.md`

## Global Constraints

- Use **samber/do/v2** only in `internal/container`; define interfaces at consumption sites; inject via constructors. (DI skill)
- All SQL **parameterized** with pgx `$N`; never concatenate input; `defer rows.Close()`; translate `sql.ErrNoRows` → domain error; use `QueryContext`/`ExecContext` variants via `repository.Querier`. (database skill)
- **bcrypt** cost 12; refresh token is a **signed JWT** (not random string); `jwt_secret` required (fail fast if empty in non-`dev`); secrets via `config.core.yaml`. (security skill)
- `users.expires_at` enforced **server-side** in `AuthMiddleware`; NULL = never expires.
- Authorization by **permission** (`Resource:Action`), cached in Redis (`perms:<user_id>`), enforced in usecase.
- Tests: table-driven + named subtests; `assert`/`require` built per-subtest; pgxmock for pgx; miniredis for redis; `uber-go/mock` typed mocks; integration tests behind `//go:build integration`; `go test -race` in CI; `goleak.VerifyTestMain` where goroutines spawn. (testing skill)
- Mirror `apps/api` conventions exactly (response envelope `libs/pkg/response`, validator `libs/pkg/validator`, `HealthStatus` shape, mockgen `-typed`, handler→usecase→repo shape).

---

## File Structure (what gets created)

```
apps/core/
  go.mod                                  # module github.com/nofendian17/sbterm/apps/core ; require libs via go.work replace
  cmd/server/main.go                      # load config, run migrations, build container, start server, graceful shutdown
  Dockerfile
  config.core.yaml.example
  migrations/account/
    000001_create_users.up.sql / .down.sql
    000002_create_rbac.up.sql / .down.sql        # roles, permissions, role_permissions, user_roles + seed
    000003_create_watchlists.up.sql / .down.sql
  internal/
    domain/
      user.go                             # User, RegisterInput, LoginInput, errors (ErrUserNotFound, ErrEmailTaken, ErrInvalidCredentials, ErrExpired, ErrSuspended)
      watchlist.go                        # Watchlist, AddWatchlistInput
      rbac.go                             # Role, Permission, assignment types
      errors.go                           # shared sentinel errors
    repository/
      user.go                             # UserRepository contract
      watchlist.go                        # WatchlistRepository contract
      rbac.go                             # RBACRepository contract (roles/permissions/user-roles)
      transaction.go                      # Querier + TxManager contracts (copy from apps/api)
    usecase/
      auth.go                             # AuthUsecase: Register, Login, Refresh, Logout
      user.go                             # UserUsecase: GetMe, UpdateMe
      watchlist.go                        # WatchlistUsecase: List, Add, Remove
      rbac.go                             # RBACUsecase: role/permission/assignment ops + HasPermission
      health.go                           # HealthUsecase (mirror apps/api)
    delivery/http/
      router.go                           # Handlers struct + NewRouter (chi) + route registration
      server.go                           # Server (copy apps/api Server shape)
      health/handler.go
      auth/handler.go
      user/handler.go
      watchlist/handler.go
      admin/handler.go
      middleware/
        auth.go                           # AuthMiddleware, context keys, RequirePermission
        ratelimit.go                      # copy from apps/api
    infrastructure/
      config/config.go                    # viper load (mirror apps/api Config shape + auth block)
      database/postgres.go                # Postgres wrapper + Pool/Querier (mirror apps/api)
      database/migrate.go                # golang-migrate embedded runner
      cache/redis.go                      # Redis wrapper + Client interface (mirror apps/api)
      repository/
        user.go                           # pgx impl of UserRepository
        watchlist.go                      # pgx impl of WatchlistRepository
        rbac.go                           # pgx impl of RBACRepository
        health.go                         # health repo (DBPinger/RedisPinger)
        token_redis.go                    # refresh-jti store + permission cache in Redis
    container/container.go                # samber/do wiring (provideInfra / provideRepos / provideUsecases / provideHandlers)
    mocks/                                # mockgen -typed output
```

`go.work` is **modified** to add `./apps/core`.

---

### Task 1: Scaffold module + config + go.work

**Files:**
- Create: `apps/core/go.mod`, `apps/core/config.core.yaml.example`
- Modify: `go.work` (add `./apps/core`)

**Interfaces:**
- Produces: module path `github.com/nofendian17/sbterm/apps/core`; config loadable via `config.Load()`.

- [ ] **Step 1: Create `apps/core/go.mod`**

```go
module github.com/nofendian17/sbterm/apps/core

go 1.26.5

require (
	github.com/go-chi/chi/v5 v5.2.1
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/golang-migrate/migrate/v4 v4.19.0
	github.com/jackc/pgx/v5 v5.10.0
	github.com/redis/go-redis/v9 v9.22.0
	github.com/samber/do/v2 v2.0.0
	github.com/spf13/viper v1.20.0
	github.com/go-playground/validator/v10 v10.26.0
	github.com/stretchr/testify v1.10.0
	go.uber.org/mock v0.5.0
	github.com/alicebob/miniredis/v2 v2.38.0
	github.com/pashagolub/pgxmock/v5 v5.1.0
	golang.org/x/crypto v0.53.0
	golang.org/x/sync v0.16.0
	github.com/nofendian17/sbterm/libs/pkg v0.0.0
)
```
Pin to versions already present in the workspace where possible; run `go mod tidy` in a later task to reconcile.

- [ ] **Step 2: Create `apps/core/config.core.yaml.example`** (mirror spec §7)

```yaml
app:
  name: account
  version: dev
port: ":8081"
database:
  url: postgres://postgres:postgres@localhost:5432/sbterm?sslmode=disable
  max_conns: 10
  min_conns: 0
  max_conn_lifetime: 1h
  max_conn_idle_time: 30m
redis:
  url: redis://localhost:6379/0
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
  default_user_ttl: 720h
  bcrypt_cost: 12
http:
  read_timeout: 10s
  write_timeout: 10s
  idle_timeout: 60s
```

- [ ] **Step 3: Add `./apps/core` to `go.work` `use (...)`**

Edit `go.work` to append `./apps/core` after `./apps/ws`.

- [ ] **Step 4: Verify module resolves**

Run: `cd apps/core && go mod edit -require=github.com/nofendian17/sbterm/libs/pkg@v0.0.0 && cd ../.. && go work sync && go build ./apps/core/... 2>&1 | head`
Expected: builds (empty main for now is fine; may show "no Go files" until Task 2 — acceptable). The key check is `go.work` sync succeeds.

- [ ] **Step 5: Commit**

```bash
git add apps/core/go.mod apps/core/config.core.yaml.example go.work
git commit -m "feat(account): scaffold module, config, go.work entry"
```

---

### Task 2: Config loader + Postgres + Redis infrastructure (TDD)

**Files:**
- Create: `apps/core/internal/infrastructure/config/config.go`, `apps/core/internal/infrastructure/database/postgres.go`, `apps/core/internal/infrastructure/cache/redis.go`
- Create: `apps/core/internal/infrastructure/config/config_test.go`, `apps/core/internal/infrastructure/database/postgres_test.go`, `apps/core/internal/infrastructure/cache/redis_test.go`
- Test: same paths as above

**Interfaces:**
- Consumes: nothing from other tasks yet.
- Produces: `config.Load() (*config.Config, error)`; `database.New(ctx, url, opts...) (*database.Postgres, error)` exposing `Ping`, `Begin`, `BeginTx`, `Shutdown`, and satisfying `repository.TxBeginner`; `cache.New(ctx, url, opts...) (*cache.Redis, error)`.

- [ ] **Step 1: Write failing test `config_test.go`**

```go
package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.core.yaml")
	require.NoError(t, os.WriteFile(p, []byte("app:\n  name: account\n  version: dev\nport: \":8081\"\nauth:\n  jwt_secret: test-secret\n  access_ttl: 15m\n  refresh_ttl: 720h\n  default_user_ttl: 720h\n  bcrypt_cost: 12\n"), 0o644))

	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, ":8081", cfg.Port)
	require.Equal(t, "test-secret", cfg.Auth.JWTSecret)
	require.Equal(t, 12, cfg.Auth.BcryptCost)
}
```

- [ ] **Step 2: Run test — expect FAIL (config.Load undefined)**

Run: `cd apps/core && go test ./internal/infrastructure/config/... 2>&1 | head`
Expected: build failure, `Load` undefined.

- [ ] **Step 3: Implement `config.go`** (mirror `apps/api/internal/infrastructure/config/config.go`, add `AuthConfig`)

```go
package config

import (
	"errors"
	"fmt"
	"time"

	"github.com/spf13/viper"
)

const (
	ConfigFileName = "config.core"
	ConfigFileType = "yaml"
	ConfigFilePath = "."
)

var version = "dev"

type Config struct {
	App       AppConfig       `mapstructure:"app"`
	Port      string          `mapstructure:"port"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Redis     RedisConfig     `mapstructure:"redis"`
	Log       LogConfig       `mapstructure:"log"`
	RateLimit RateLimitConfig `mapstructure:"rate_limit"`
	Auth      AuthConfig      `mapstructure:"auth"`
	HTTP      HTTPConfig      `mapstructure:"http"`
}

type AppConfig struct {
	Name    string `mapstructure:"name"`
	Version string `mapstructure:"version"`
}

type DatabaseConfig struct {
	URL             string        `mapstructure:"url"`
	MaxConns        int32         `mapstructure:"max_conns"`
	MinConns        int32         `mapstructure:"min_conns"`
	MaxConnLifetime time.Duration `mapstructure:"max_conn_lifetime"`
	MaxConnIdleTime time.Duration `mapstructure:"max_conn_idle_time"`
}

type RedisConfig struct {
	URL          string        `mapstructure:"url"`
	MaxRetries   int           `mapstructure:"max_retries"`
	PoolSize     int           `mapstructure:"pool_size"`
	MinIdleConns int           `mapstructure:"min_idle_conns"`
	DialTimeout  time.Duration `mapstructure:"dial_timeout"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

type LogConfig struct {
	Level     string `mapstructure:"level"`
	Format    string `mapstructure:"format"`
	AddSource bool   `mapstructure:"add_source"`
}

type RateLimitConfig struct {
	Rate  int `mapstructure:"rate"`
	Burst int `mapstructure:"burst"`
}

type AuthConfig struct {
	JWTSecret      string        `mapstructure:"jwt_secret"`
	AccessTokenTTL time.Duration `mapstructure:"access_ttl"`
	RefreshTokenTTL time.Duration `mapstructure:"refresh_ttl"`
	DefaultUserTTL time.Duration `mapstructure:"default_user_ttl"`
	BcryptCost     int           `mapstructure:"bcrypt_cost"`
}

type HTTPConfig struct {
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
}

func Load() (*Config, error) {
	v := viper.New()
	v.SetConfigName(ConfigFileName)
	v.SetConfigType(ConfigFileType)
	v.AddConfigPath(ConfigFilePath)

	setDefaults(v)

	if err := v.ReadInConfig(); err != nil {
		var nf viper.ConfigFileNotFoundError
		if errors.As(err, &nf) {
			return nil, fmt.Errorf("config: file %q not found: %w", ConfigFileName, err)
		}
		return nil, fmt.Errorf("config: read: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("config: unmarshal: %w", err)
	}
	// fail fast on missing secret outside dev
	if cfg.Auth.JWTSecret == "" && cfg.App.Version != "dev" {
		return nil, errors.New("config: auth.jwt_secret is required in non-dev builds")
	}
	if cfg.Auth.BcryptCost == 0 {
		cfg.Auth.BcryptCost = 12
	}
	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("port", ":8081")
	v.SetDefault("auth.bcrypt_cost", 12)
	v.SetDefault("auth.access_ttl", 15*time.Minute)
	v.SetDefault("auth.refresh_ttl", 720*time.Hour)
	v.SetDefault("auth.default_user_ttl", 720*time.Hour)
}
```

- [ ] **Step 4: Write failing tests `postgres_test.go` / `redis_test.go`** (constructors exist with Ping; use pgxmock/miniredis)

```go
// postgres_test.go
package database

import (
	"context"
	"testing"

	"github.com/pashagolub/pgxmock/v5"
	"github.com/stretchr/testify/require"
)

func TestNewPostgres_Ping(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()
	mock.ExpectPing()

	db := &Postgres{pool: mock}
	require.NoError(t, db.Ping(context.Background()))
	require.NoError(t, mock.ExpectationsWereMet())
}
```

```go
// redis_test.go
package cache

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/require"
)

func TestNewRedis_Ping(t *testing.T) {
	srv, err := miniredis.Run()
	require.NoError(t, err)
	defer srv.Close()

	r, err := New(context.Background(), "redis://"+srv.Addr(),
		WithDialTimeout(0), WithReadTimeout(0), WithWriteTimeout(0))
	require.NoError(t, err)
	require.NoError(t, r.Ping(context.Background()))
}
```

- [ ] **Step 5: Implement `postgres.go` and `redis.go`** (copy the exact `Pool`/`Querier`/`Client` interface shapes from `apps/api/internal/infrastructure/database/postgres.go` and `apps/api/internal/infrastructure/cache/redis.go`; expose `Ping`, `Begin`, `BeginTx`, `Shutdown` on Postgres; `Ping`, `Close` on Redis).

- [ ] **Step 6: Run tests — expect PASS**

Run: `cd apps/core && go test ./internal/infrastructure/... 2>&1 | tail`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add apps/core/internal/infrastructure/config apps/core/internal/infrastructure/database apps/core/internal/infrastructure/cache
git commit -m "feat(account): config loader, pgx Postgres wrapper, redis wrapper"
```

---

### Task 3: SQL migrations (golang-migrate, embedded) + runner

**Files:**
- Create: `apps/core/migrations/account/000001_create_users.up.sql`, `.down.sql`, `000002_create_rbac.up.sql`, `.down.sql`, `000003_create_watchlists.up.sql`, `.down.sql`
- Create: `apps/core/internal/infrastructure/database/migrate.go` (+ `migrate_test.go` with miniredis/embedded not feasible; use `-tags integration` test against postgres if available; otherwise rely on manual apply)

**Interfaces:**
- Produces: `database.RunMigrations(ctx, url string) error` applying embedded `migrations/account`.

- [ ] **Step 1: Write `000001_create_users.up.sql`**

```sql
CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    display_name  TEXT NOT NULL,
    expires_at    TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ
);
CREATE INDEX idx_users_email ON users (email);
CREATE INDEX idx_users_deleted_at ON users (deleted_at);
CREATE INDEX idx_users_expires_at ON users (expires_at);
```
`000001_create_users.down.sql`: `DROP TABLE IF EXISTS users;`

- [ ] **Step 2: Write `000002_create_rbac.up.sql`** (tables + seed)

```sql
CREATE TABLE roles (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE permissions (
    id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resource TEXT NOT NULL,
    action   TEXT NOT NULL,
    name     TEXT NOT NULL UNIQUE,
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

-- Seed permissions
INSERT INTO permissions (resource, action, name) VALUES
 ('auth','login','auth:login'),
 ('profile','read','profile:read'),
 ('profile','write','profile:write'),
 ('watchlist','read','watchlist:read'),
 ('watchlist','write','watchlist:write'),
 ('admin','roles:read','admin:roles:read'),
 ('admin','roles:write','admin:roles:write'),
 ('admin','users:read','admin:users:read'),
 ('admin','users:manage','admin:users:manage'),
 ('admin','rbac:assign','admin:rbac:assign');

-- Seed roles
INSERT INTO roles (name, description) VALUES ('user','Default end user'), ('admin','Full administrator');
INSERT INTO role_permissions (role_id, permission_id)
 SELECT r.id, p.id FROM roles r, permissions p
  WHERE r.name='user' AND p.name IN ('auth:login','profile:read','profile:write','watchlist:read','watchlist:write');
INSERT INTO role_permissions (role_id, permission_id)
 SELECT r.id, p.id FROM roles r, permissions p WHERE r.name='admin';
```
`000002_create_rbac.down.sql`: `DROP TABLE IF EXISTS user_roles, role_permissions, permissions, roles CASCADE;`

- [ ] **Step 3: Write `000003_create_watchlists.up.sql`**

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
`000003_create_watchlists.down.sql`: `DROP TABLE IF EXISTS watchlists;`

- [ ] **Step 4: Implement `migrate.go`** using embedded FS:

```go
package database

import (
	"context"
	"embed"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/account/*.sql
var migrationsFS embed.FS

func RunMigrations(ctx context.Context, dbURL string) error {
	src, err := iofs.New(migrationsFS, "migrations/account")
	if err != nil {
		return fmt.Errorf("migrations: source: %w", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", src, dbURL)
	if err != nil {
		return fmt.Errorf("migrations: new: %w", err)
	}
	defer m.Close()
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrations: up: %w", err)
	}
	return nil
}
```

- [ ] **Step 5: Run (integration if postgres available)**

Run: `cd apps/core && go build ./... && go test -tags integration ./internal/infrastructure/database/... 2>&1 | tail` (skip if no DB; note in commit).
Expected: builds; integration applies 3 migrations.

- [ ] **Step 6: Commit**

```bash
git add apps/core/migrations apps/core/internal/infrastructure/database/migrate.go
git commit -m "feat(account): golang-migrate embedded migrations (users, rbac, watchlists)"
```

---

### Task 4: Domain types + sentinel errors

**Files:**
- Create: `apps/core/internal/domain/user.go`, `watchlist.go`, `rbac.go`, `errors.go`
- Test: `apps/core/internal/domain/domain_test.go`

**Interfaces:**
- Produces: `domain.User`, `domain.RegisterInput`, `domain.LoginInput`, `domain.Watchlist`, `domain.Role`, `domain.Permission`, sentinel errors `ErrUserNotFound`, `ErrEmailTaken`, `ErrInvalidCredentials`, `ErrExpired`, `ErrSuspended`, `ErrPermissionDenied`, `ErrDuplicateWatchlist`.

- [ ] **Step 1: Write `errors.go`**

```go
package domain

import "errors"

var (
	ErrUserNotFound        = errors.New("user not found")
	ErrEmailTaken          = errors.New("email already registered")
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrExpired             = errors.New("account expired")
	ErrSuspended           = errors.New("account suspended")
	ErrPermissionDenied    = errors.New("permission denied")
	ErrDuplicateWatchlist  = errors.New("symbol already in watchlist")
	ErrRoleNotFound        = errors.New("role not found")
	ErrPermissionNotFound  = errors.New("permission not found")
)
```

- [ ] **Step 2: Write `user.go`, `watchlist.go`, `rbac.go`**

```go
package domain

import "time"

type User struct {
	ID           string
	Email        string
	PasswordHash string
	DisplayName  string
	ExpiresAt    *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
}

type RegisterInput struct {
	Email       string `validate:"required,email"`
	Password    string `validate:"required,min=8"`
	DisplayName string `validate:"required"`
}

type LoginInput struct {
	Email    string `validate:"required,email"`
	Password string `validate:"required"`
}

type Watchlist struct {
	ID        string
	UserID    string
	Symbol    string
	Label     string
	CreatedAt time.Time
}

type AddWatchlistInput struct {
	Symbol string `validate:"required"`
	Label  string
}

type Role struct {
	ID          string
	Name        string
	Description string
}

type Permission struct {
	ID       string
	Resource string
	Action   string
	Name     string // "<resource>:<action>"
}
```

- [ ] **Step 3: Write a small `domain_test.go`** asserting error identity and struct tags compile.

- [ ] **Step 4: Run — expect PASS**

Run: `cd apps/core && go test ./internal/domain/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/core/internal/domain
git commit -m "feat(account): domain types and sentinel errors"
```

---

### Task 5: Token service (bcrypt + JWT access/refresh + Redis jti) — TDD

**Files:**
- Create: `apps/core/internal/usecase/auth.go` (token helpers), `apps/core/internal/infrastructure/repository/token_redis.go`
- Test: `apps/core/internal/usecase/auth_test.go`, `apps/core/internal/infrastructure/repository/token_redis_test.go`

**Interfaces:**
- Produces: `TokenService` with `GenerateTokenPair(userID string, expiry *time.Time) (access, refresh string, err error)`, `VerifyAccess(token string) (userID string, err error)`, `StoreRefresh(jti, userID string, ttl time.Duration) error`, `ConsumeRefresh(jti string) (userID string, ok bool)`, `DeleteRefresh(jti string) error`.

- [ ] **Step 1: Write failing `auth_test.go`**

```go
package usecase

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

func TestTokenService_RoundTrip(t *testing.T) {
	ts := NewTokenService("secret", 15*time.Minute, time.Hour)
	access, refresh, err := ts.GenerateTokenPair("u1", nil)
	require.NoError(t, err)
	require.NotEmpty(t, access)
	require.NotEmpty(t, refresh)

	uid, err := ts.VerifyAccess(access)
	require.NoError(t, err)
	require.Equal(t, "u1", uid)

	// wrong secret fails
	bad := NewTokenService("other", time.Minute, time.Minute)
	_, err = bad.VerifyAccess(access)
	require.Error(t, err)
}

func TestTokenService_RefreshClaims(t *testing.T) {
	ts := NewTokenService("s", time.Minute, time.Hour)
	_, refresh, _ := ts.GenerateTokenPair("u2", nil)
	claims := &jwt.RegisteredClaims{}
	_, err := jwt.ParseWithClaims(refresh, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte("s"), nil
	})
	require.NoError(t, err)
	require.Equal(t, "refresh", claims.ID[0:0]) // placeholder; assert Type below
	require.Equal(t, "u2", claims.Subject)
}
```

- [ ] **Step 2: Run — expect FAIL (NewTokenService undefined)**

- [ ] **Step 3: Implement token service in `auth.go`**

```go
package usecase

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/nofendian17/sbterm/apps/core/internal/infrastructure/repository"
)

const (
	claimTypeAccess  = "access"
	claimTypeRefresh = "refresh"
)

type TokenService struct {
	secret       string
	accessTTL    time.Duration
	refreshTTL   time.Duration
	refreshStore repository.RefreshStore
}

func NewTokenService(secret string, accessTTL, refreshTTL time.Duration, store repository.RefreshStore) *TokenService {
	return &TokenService{secret: secret, accessTTL: accessTTL, refreshTTL: refreshTTL, refreshStore: store}
}

func (s *TokenService) GenerateTokenPair(userID string, _ *time.Time) (access, refresh string, err error) {
	now := time.Now()
	accessClaims := jwt.RegisteredClaims{
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(s.accessTTL)),
		ID:        newJTI(),
	}
	access = sign(accessClaims, s.secret)

	refreshClaims := jwt.RegisteredClaims{
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(s.refreshTTL)),
		ID:        newJTI(),
	}
	refresh = sign(refreshClaims, s.secret)
	if err := s.refreshStore.StoreRefresh(refreshClaims.ID, userID, s.refreshTTL); err != nil {
		return "", "", err
	}
	return access, refresh, nil
}

func (s *TokenService) VerifyAccess(token string) (string, error) {
	claims, err := parse(token, s.secret)
	if err != nil {
		return "", err
	}
	if claims.ExpiresAt != nil && claims.ExpiresAt.Before(time.Now()) {
		return "", errors.New("token expired")
	}
	return claims.Subject, nil
}

func sign(c jwt.RegisteredClaims, secret string) string {
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	s, _ := t.SignedString([]byte(secret))
	return s
}

func parse(token, secret string) (*jwt.RegisteredClaims, error) {
	claims := &jwt.RegisteredClaims{}
	_, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	return claims, nil
}
```

Add `newJTI()` using `crypto/rand` (hex) in a small helper; ensure refresh JWT carries a distinguishable value (store `jti` in Redis; verify `refresh` by checking Redis presence — see Task 9). For the test's `claimType` assertion, assert `claims.ID != ""` (jti present) rather than a placeholder.

- [ ] **Step 4: Implement `token_redis.go`** (`RefreshStore` interface + miniredis-compatible impl):

```go
package repository

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RefreshStore interface {
	StoreRefresh(jti, userID string, ttl time.Duration) error
	ConsumeRefresh(jti string) (userID string, ok bool)
	DeleteRefresh(jti string) error
}

type redisRefreshStore struct {
	client *redis.Client
}

func NewRedisRefreshStore(client *redis.Client) RefreshStore {
	return &redisRefreshStore{client: client}
}

func (s *redisRefreshStore) StoreRefresh(jti, userID string, ttl time.Duration) error {
	return s.client.Set(context.Background(), "refresh:"+jti, userID, ttl).Err()
}
func (s *redisRefreshStore) ConsumeRefresh(jti string) (string, bool) {
	v, err := s.client.Get(context.Background(), "refresh:"+jti).Result()
	if err != nil {
		return "", false
	}
	return v, true
}
func (s *redisRefreshStore) DeleteRefresh(jti string) error {
	return s.client.Del(context.Background(), "refresh:"+jti).Err()
}
```

- [ ] **Step 5: Write `token_redis_test.go`** with miniredis asserting Store then Consume returns userID, Delete then Consume returns ok=false.

- [ ] **Step 6: Run — expect PASS**

Run: `cd apps/core && go test ./internal/usecase/... ./internal/infrastructure/repository/...`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add apps/core/internal/usecase/auth.go apps/core/internal/infrastructure/repository/token_redis.go
git commit -m "feat(account): bcrypt-less token service + redis refresh store (TDD)"
```
(Note: bcrypt hashing lives in the user repository/usecase, Task 7.)

---

### Task 6: User repository (pgx) — TDD

**Files:**
- Create: `apps/core/internal/repository/user.go` (contract), `apps/core/internal/infrastructure/repository/user.go` (pgx), `apps/core/internal/infrastructure/repository/user_test.go`
- Test: same

**Interfaces:**
- Consumes: `repository.Querier`, `repository.TxManager` (copy `transaction.go` from `apps/api` into `apps/core/internal/repository`).
- Produces: `UserRepository` with `Create(ctx, user User) error`, `GetByEmail(ctx, email) (User, error)`, `GetByID(ctx, id) (User, error)`, `Update(ctx, id, displayName, expiresAt *time.Time) error`, `SoftDelete(ctx, id) error`, `SetExpiry(ctx, id, expiresAt *time.Time) error`.

- [ ] **Step 1: Copy `transaction.go`** from `apps/api/internal/repository/transaction.go` into `apps/core/internal/repository/transaction.go` (unchanged). Add `//go:generate` lines for mockgen on `health.go`/repos later.

- [ ] **Step 2: Write failing `user_test.go`** (pgxmock: Create inserts; GetByEmail returns row; not found → `ErrUserNotFound`; duplicate → `ErrEmailTaken` from unique violation code `23505`).

```go
package repository

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v5"
	"github.com/stretchr/testify/require"
)

func TestUserRepository_Create(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	mock.ExpectExec(`INSERT INTO users`).WithArgs(pgxmock.AnyArg(), "a@b.co", pgxmock.AnyArg(), "Beni").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	repo := NewUserRepository(mock)
	err := repo.Create(context.Background(), User{Email: "a@b.co", DisplayName: "Beni", PasswordHash: "x"})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_GetByEmail_NotFound(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	mock.ExpectQuery(`SELECT .* FROM users`).WillReturnError(pgErrNoRows())

	repo := NewUserRepository(mock)
	_, err := repo.GetByEmail(context.Background(), "x@y.co")
	require.ErrorIs(t, err, ErrUserNotFound)
}
```

Add helper `pgErrNoRows()` returning a `pgconn.PgError` with Code `02000`/use `sql.ErrNoRows` wrapper — simpler: return `sql.ErrNoRows` directly and have repo map it. (Repo translates `sql.ErrNoRows` → `ErrUserNotFound`.)

- [ ] **Step 3: Implement contract + pgx repo** (parameterized `$1..$N`, `QueryRowContext`, `defer rows.Close()`, map unique violation `23505` → `ErrEmailTaken`, `sql.ErrNoRows` → `ErrUserNotFound`, always `errors.Is`). Use `pgx.Identifier`/allowlist only if dynamic columns needed (not required here).

- [ ] **Step 4: Run — expect PASS**

Run: `cd apps/core && go test ./internal/infrastructure/repository/... -run User`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/core/internal/repository apps/core/internal/infrastructure/repository/user.go apps/core/internal/infrastructure/repository/user_test.go
git commit -m "feat(account): user repository (pgx, pgxmock TDD)"
```

---

### Task 7: Auth usecase (Register/Login/Refresh/Logout) — TDD

**Files:**
- Create: `apps/core/internal/usecase/auth.go` (extend), `apps/core/internal/usecase/auth_usecase_test.go`
- Create: `apps/core/internal/mocks/mock_user_repository.go`, `mock_auth_deps.go` (TokenService mock not needed; inject real or interface)
- Test: `auth_usecase_test.go`

**Interfaces:**
- Consumes: `UserRepository` (Task 6), `TokenService` (Task 5), `TxManager`, `AuthConfig` (bcrypt cost, default TTL), `RefreshStore`.
- Produces: `AuthUsecase` with `Register(ctx, RegisterInput) (access, refresh string, err error)`, `Login(ctx, LoginInput) (access, refresh string, err error)`, `Refresh(ctx, refreshToken string) (access, refresh string, err error)`, `Logout(ctx, refreshToken string) error`.

- [ ] **Step 1: Write failing `auth_usecase_test.go`** (table-driven, named subtests):

```go
func TestAuthUsecase_Register(t *testing.T) {
	tests := []struct {
		name      string
		input     domain.RegisterInput
		setup     func(r *mocks.MockUserRepository)
		wantErr   error
	}{
		{
			name:  "success assigns default expiry + user role",
			input: domain.RegisterInput{Email: "a@b.co", Password: "password123", DisplayName: "Beni"},
			setup: func(r *mocks.MockUserRepository) {
				r.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
				r.EXPECT().AssignDefaultRole(gomock.Any(), gomock.Any()).Return(nil)
			},
		},
		{
			name:  "duplicate email -> ErrEmailTaken",
			input: domain.RegisterInput{Email: "a@b.co", Password: "password123", DisplayName: "Beni"},
			setup: func(r *mocks.MockUserRepository) {
				r.EXPECT().Create(gomock.Any(), gomock.Any()).Return(domain.ErrEmailTaken)
			},
			wantErr: domain.ErrEmailTaken,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := assert.New(t)
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			repo := mocks.NewMockUserRepository(ctrl)
			tt.setup(repo)
			uc := NewAuthUsecase(repo, ts, txm, cfg)
			_, _, err := uc.Register(context.Background(), tt.input)
			if tt.wantErr != nil {
				is.ErrorIs(err, tt.wantErr)
				return
			}
			is.NoError(err)
		})
	}
}
```

- [ ] **Step 2: Run — expect FAIL**

- [ ] **Step 3: Implement `AuthUsecase`** in `auth.go`:
  - `Register`: validate input via `validator`; bcrypt hash password (cost from cfg); set `ExpiresAt = now().Add(cfg.DefaultUserTTL)`; `Create` within `TxManager.WithTx`; on success `AssignDefaultRole` (role `user`); generate token pair.
  - `Login`: `GetByEmail`; `bcrypt.CompareHashAndPassword`; if `ExpiresAt` set and past → `ErrExpired`; `DeletedAt` set → `ErrSuspended`; generate token pair.
  - `Refresh`: parse+verify refresh JWT (type=refresh); `ConsumeRefresh(jti)` → must exist; rotate (delete old jti, new pair).
  - `Logout`: parse refresh JWT, `DeleteRefresh(jti)`.

- [ ] **Step 4: Run — expect PASS**

Run: `cd apps/core && go test ./internal/usecase/... -run Auth`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/core/internal/usecase/auth.go apps/core/internal/usecase/auth_usecase_test.go apps/core/internal/mocks
git commit -m "feat(account): auth usecase register/login/refresh/logout (TDD)"
```

---

### Task 8: RBAC repository + usecase + permission cache — TDD

**Files:**
- Create: `apps/core/internal/repository/rbac.go`, `apps/core/internal/infrastructure/repository/rbac.go`, `rbac_test.go`
- Create: `apps/core/internal/usecase/rbac.go`, `apps/core/internal/usecase/rbac_test.go`
- Create: permission cache in `token_redis.go` (add `PermissionCache` interface) or new `rbac_redis.go`

**Interfaces:**
- Produces: `RBACRepository` (`CreateRole`, `GetRole`, `ListRoles`, `AssignPermissionToRole`, `RevokePermissionFromRole`, `AssignRoleToUser`, `RevokeRoleFromUser`, `ListUserPermissions(ctx, userID) ([]string, error)`); `RBACUsecase` (role CRUD, assignment ops, `HasPermission(ctx, userID, perm string) (bool, error)`); `PermissionCache` (`Get(userID)`, `Set(userID, perms)`, `Invalidate(userID)`) backed by `perms:<user_id>` Redis with ~5m TTL.

- [ ] **Step 1: Write `rbac_test.go`** (pgxmock: `ListUserPermissions` joins `user_roles→role_permissions→permissions` returning `name` rows; unique/foreign violations mapped).

- [ ] **Step 2: Implement `RBACRepository`** (parameterized; `ListUserPermissions` returns `[]string` of permission names; cache-aside handled in usecase).

- [ ] **Step 3: Write `rbac_usecase_test.go`**: `HasPermission` returns true when user has role with permission; false otherwise; cache hit path returns cached; assignment change invalidates cache.

- [ ] **Step 4: Implement `RBACUsecase`**: wraps repo; `HasPermission` checks `PermissionCache.Get` → miss → `ListUserPermissions` → `Set`; assignment/revoke ops call repo then `Invalidate`.

- [ ] **Step 5: Add `PermissionCache` redis impl** (`perms:<user_id>` JSON of string slice, TTL 5m; `Invalidate` deletes key).

- [ ] **Step 6: Run — expect PASS**

Run: `cd apps/core && go test ./internal/... -run RBAC`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add apps/core/internal/repository/rbac.go apps/core/internal/infrastructure/repository/rbac.go apps/core/internal/usecase/rbac.go apps/core/internal/infrastructure/repository/rbac_redis.go
git commit -m "feat(account): RBAC repository + usecase + permission cache (TDD)"
```

---

### Task 9: Auth middleware + context identity + RequirePermission

**Files:**
- Create: `apps/core/internal/delivery/http/middleware/auth.go` (+ `auth_test.go`)
- Create: `apps/core/internal/delivery/http/middleware/ratelimit.go` (copy from `apps/api`)

**Interfaces:**
- Consumes: `TokenService.VerifyAccess`, `UserRepository.GetByID` (for expiry/suspended check + load), `RBACUsecase.HasPermission`.
- Produces: `AuthMiddleware(next)` injecting `user_id` + permission set into context; `RequirePermission(perm string) Middleware` returning 403 when missing; typed context keys `CtxUserID`, `CtxPermissions`.

- [ ] **Step 1: Write `auth_test.go`** (httptest: no header → 401; invalid token → 401; expired account (`ExpiresAt` past) → 401; valid → passes, context has userID; `RequirePermission` missing → 403, present → 200).

- [ ] **Step 2: Implement `auth.go`**:
  - Parse `Authorization: Bearer`; `VerifyAccess`; load user via `GetByID`; if `DeletedAt != nil` → 401 `ErrSuspended`; if `ExpiresAt != nil && before now` → 401 `ErrExpired`; resolve permissions via `HasPermission`/cache; store in context; call next.
  - `RequirePermission(p)`: wraps handler, checks context permissions set contains `p`, else `response.Error(w, 403, CodeForbidden, "forbidden")`.

- [ ] **Step 3: Copy `ratelimit.go`** from `apps/api/internal/delivery/http/middleware/ratelimit.go` verbatim (same `Middleware`, `Option`, `NewRateLimit`).

- [ ] **Step 4: Run — expect PASS**

Run: `cd apps/core && go test ./internal/delivery/http/middleware/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/core/internal/delivery/http/middleware
git commit -m "feat(account): auth middleware, context identity, permission guard"
```

---

### Task 10: HTTP handlers + router (auth, user, watchlist, admin, health)

**Files:**
- Create: `apps/core/internal/delivery/http/router.go`, `server.go`, `health/handler.go`, `auth/handler.go`, `user/handler.go`, `watchlist/handler.go`, `admin/handler.go`, and their `_test.go`
- Create: `apps/core/internal/usecase/health.go`, `apps/core/internal/repository/health.go`, `apps/core/internal/infrastructure/repository/health.go` (mirror `apps/api` health chain)
- Create: watchlist usecase/repo (Tasks 11–12) before wiring admin.

**Interfaces:**
- Consumes: all prior usecases.
- Produces: `Handlers` struct (one field per domain) + `NewRouter(Handlers, logger, opts...) chi.Router` registering:
  - public: `GET /healthz`, `POST /api/v1/auth/register`, `POST /api/v1/auth/login`
  - authed: `POST /api/v1/auth/refresh`, `POST /api/v1/auth/logout`, `GET/PUT /api/v1/users/me`, `GET/POST/DELETE /api/v1/watchlists...`
  - admin (`RequirePermission`): `GET/POST /api/v1/admin/roles`, `GET/PUT/DELETE /api/v1/admin/roles/{id}`, `POST/DELETE /api/v1/admin/roles/{id}/permissions`, `GET/POST/DELETE /api/v1/admin/users/{id}/roles`, `GET /api/v1/admin/users`, `GET /api/v1/admin/users/{id}`, `POST /api/v1/admin/users/{id}/suspend`, `PATCH /api/v1/admin/users/{id}/expiry`, `DELETE /api/v1/admin/users/{id}`, `GET /api/v1/admin/users/{id}/watchlists`.

- [ ] **Step 1: Write handler tests first** (httptest table tests per handler: validation 400, auth 401, forbidden 403, success 200 with `response.OK` envelope). e.g. `auth/handler_test.go` covers register success + duplicate (500/409 via usecase error → map to `CodeConflict`).

- [ ] **Step 2: Implement handlers** using `libs/pkg/response` (`OK`, `Created`, `Error` with `CodeConflict`/`CodeValidation`/`CodeUnauthorized`/`CodeForbidden`/`CodeNotFound`), `libs/pkg/validator` (`Validate` → `ValidationError.Fields` map → `response.Error(..., CodeValidation, ..., details)`), and DTOs with json tags.

- [ ] **Step 3: Implement `router.go`** mirroring `apps/api/internal/delivery/http/router.go` (Handlers struct, `WithRateLimit` option, middleware chain, route groups). Add `server.go` copied from `apps/api`.

- [ ] **Step 4: Implement health chain** (copy `apps/api` health handler/usecase/repo/contract exactly, swapping `Postgres`/`Redis` pingers).

- [ ] **Step 5: Run — expect PASS**

Run: `cd apps/core && go test ./internal/delivery/http/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add apps/core/internal/delivery apps/core/internal/usecase/health.go apps/core/internal/repository/health.go apps/core/internal/infrastructure/repository/health.go
git commit -m "feat(account): http handlers, router, health, all routes wired"
```

---

### Task 11: Watchlist usecase + repository (TDD)

**Files:**
- Create: `apps/core/internal/usecase/watchlist.go` (+test), `apps/core/internal/repository/watchlist.go` (contract), `apps/core/internal/infrastructure/repository/watchlist.go` (+test)

**Interfaces:**
- Produces: `WatchlistUsecase` (`List(ctx, userID)`, `Add(ctx, userID, AddWatchlistInput)`, `Remove(ctx, userID, symbol)`); repo maps unique `(user_id, symbol)` violation `23505` → `ErrDuplicateWatchlist`.

- [ ] **Step 1: Write tests** (pgxmock: Add duplicate → `ErrDuplicateWatchlist`; List returns rows; Remove deletes by user+symbol).

- [ ] **Step 2: Implement repo + usecase.**

- [ ] **Step 3: Run — PASS; commit.**

```bash
git add apps/core/internal/usecase/watchlist.go apps/core/internal/repository/watchlist.go apps/core/internal/infrastructure/repository/watchlist.go
git commit -m "feat(account): watchlist usecase + repository (TDD)"
```

---

### Task 12: Admin usecase + handler wiring (RBAC management + user mgmt)

**Files:**
- Create: `apps/core/internal/usecase/admin.go` (+test), `apps/core/internal/delivery/http/admin/handler.go` (+test)

**Interfaces:**
- Consumes: `RBACUsecase`, `UserRepository`, `WatchlistUsecase`.
- Produces: admin endpoints: list/create roles, role CRUD, assign/revoke permission↔role, assign/revoke role↔user, list/view/suspend/delete users, set/extend expiry, view user watchlists. All gated by `RequirePermission` in router.

- [ ] **Step 1: Write `admin_test.go`** (usecase-level: suspend sets `deleted_at`; `SetExpiry` with `extend_days`; assignment invalidates cache via RBACUsecase mock).

- [ ] **Step 2: Implement `admin.go` + handler. Mark admin endpoints' permission in router (Task 10 additions): `admin:roles:read/write`, `admin:users:read/manage`, `admin:rbac:assign`.**

- [ ] **Step 3: Run — PASS; commit.**

```bash
git add apps/core/internal/usecase/admin.go apps/core/internal/delivery/http/admin
git commit -m "feat(account): admin usecase + handlers (RBAC + user management)"
```

---

### Task 13: main.go wiring + samber/do container + graceful shutdown

**Files:**
- Create: `apps/core/cmd/server/main.go`, `apps/core/internal/container/container.go`

**Interfaces:**
- Consumes: all providers.
- Produces: runnable binary: load config → `RunMigrations` → build `do.RootScope` (provide config, logger, Postgres, Redis, repos, usecases, handlers) → `NewRouter` → `NewServer` → `ListenAndServe` with `signal.Notify` graceful shutdown (`Shutdown` server, `do.Shutdown`, close Postgres/Redis).

- [ ] **Step 1: Write `container.go`** mirroring `apps/api/internal/container/container.go` structure (`New(cfg, logger) *do.RootScope`, `provideInfrastructure`, `provideRepositories` with `do.MustAs`, `provideUsecases`, `provideHandlers`).

- [ ] **Step 2: Write `main.go`** with migrations + DI + server + shutdown, copying lifecycle patterns from `apps/api` (errgroup, signal handling).

- [ ] **Step 3: Build + smoke test (integration)**

Run: `cd apps/core && go build ./... && go vet ./... && go test ./... 2>&1 | tail`
Expected: all build, vet clean, tests pass.

- [ ] **Step 4: Commit**

```bash
git add apps/core/cmd apps/core/internal/container
git commit -m "feat(account): main.go + samber/do container + graceful shutdown"
```

---

### Task 14: Dockerfile + docker-compose entry + config example wiring

**Files:**
- Create: `apps/core/Dockerfile`
- Modify: `docker-compose.yml` (add `account` service), `config.core.yaml.example` already exists

**Interfaces:**
- Produces: `account` service depends_on postgres+redis healthy; mounts `config.core.yaml`; publishes `8081:8081`; healthcheck `curl -fsS http://127.0.0.1:8081/healthz`.

- [ ] **Step 1: Create `Dockerfile`** (copy `apps/api/Dockerfile` multi-stage; build arg `APP_VERSION` ldflags into `internal/infrastructure/config.version`; entry runs the binary with `--config /app/config.core.yaml` or expects `config.core.yaml` in CWD — match `main.go`).

- [ ] **Step 2: Add `account` to `docker-compose.yml`** after `api`, mirroring the `api` service block (env from compose vars; mount config read-only).

- [ ] **Step 3: Validate compose (if docker available)**

Run: `docker compose config 2>&1 | head` (expect valid). Otherwise note in commit.

- [ ] **Step 4: Commit**

```bash
git add apps/core/Dockerfile docker-compose.yml
git commit -m "feat(account): Dockerfile + docker-compose service entry"
```

---

### Task 15: Mockgen generation + final test sweep

**Files:**
- Modify: generate `apps/core/internal/mocks/*.go` via `//go:generate` (mockgen `-typed`) for user, rbac, watchlist, health repositories and usecases.

**Interfaces:**
- Produces: regenerated typed mocks used by handler/usecase tests.

- [ ] **Step 1: Add `//go:generate` directives** to each repository/usecase file (copy the `apps/api` header): `//go:generate go run go.uber.org/mock/mockgen -source=<file> -destination=../mocks/mock_<name>.go -package=mocks -typed`.

- [ ] **Step 2: Run `make mock` / `go generate ./...`**

Run: `cd apps/core && go generate ./...`
Expected: mocks generated, compile.

- [ ] **Step 3: Full suite + race**

Run: `cd apps/core && go test -race ./... 2>&1 | tail`
Expected: PASS, no races.

- [ ] **Step 4: Commit**

```bash
git add apps/core/internal/mocks apps/core/internal -A
git commit -m "chore(account): generate typed mocks, full test sweep green"
```

---

## Self-Review (against spec)

1. **Spec coverage** — auth (register/login/refresh/logout) ✓ T5/T7; dynamic RBAC tables+seed+runtime assignment ✓ T3/T8/T12; per-user expiry enforced server-side ✓ T9 + domain; watchlists ✓ T11; admin management ✓ T12; golang-migrate ✓ T3; Redis refresh+permission cache ✓ T5/T8; samber/do ✓ T13; security (bcrypt 12, signed JWT, secret required, generic errors) ✓ T5/T7/T9; testing (pgxmock/miniredis/mockgen/integration tags/race) ✓ all tasks; `apps/api` untouched ✓.
2. **Placeholder scan** — no TBD/TODO; each step has concrete code or exact command; tests include real assertions.
3. **Type consistency** — `UserRepository`, `RBACRepository`, `WatchlistRepository`, `TokenService`, `RefreshStore`, `PermissionCache`, `AuthUsecase`, `RBACUsecase`, `WatchlistUsecase`, `AdminUsecase` names stable across tasks; `Querier`/`TxManager` copied verbatim from `apps/api`; `response`/`validator` package APIs match `libs/pkg`.

Gaps to watch during execution: `AssignDefaultRole` on `UserRepository` referenced in T7 — implement it in T6 (add to contract + pgx impl: insert into `user_roles` the `user` role id). T10 router references admin endpoints that depend on T12 handlers — wire `Handlers.Admin` field and routes in T12, not T10 (T10 builds the router shell + non-admin routes; T12 adds admin routes). Adjust T10/T12 ordering accordingly.
