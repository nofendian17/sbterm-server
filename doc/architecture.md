# Architecture

This document explains the `sbterm-server` architecture: layer boundaries, dependency direction, application lifecycle, and the rationale behind the main design choices.

## Overview

`sbterm-server` uses **Clean Architecture** to separate business/application rules from technical details such as HTTP, database access, cache access, configuration, logging, and dependency injection.

Main dependency direction:

```text
cmd/server
    ↓
internal/container
    ↓ wires
internal/delivery/http ──→ internal/usecase ──→ internal/repository
                                      ↓                ↑
                               internal/domain         │
                                                       │ implemented by
internal/infrastructure/repository ────────────────────┘
    ↓
internal/infrastructure/database
internal/infrastructure/cache
```

Important rules:

- Inner layers must not depend on outer layers.
- `domain`, `usecase`, and repository contracts must not depend on HTTP, database drivers, cache drivers, loggers, or the DI container.
- Technical details live in `delivery`, `infrastructure`, `pkg`, and `container`.
- `samber/do` is used only in `internal/container` as the composition root.

## Directory Structure

```text
cmd/server/
  main.go

internal/
  container/
    container.go

  domain/
    health.go

  usecase/
    health.go

  repository/
    health.go

  delivery/http/
    health.go
    router.go
    server.go
    middleware/
      ratelimit.go

  infrastructure/
    cache/
      redis.go
    config/
      config.go
    database/
      postgres.go
    repository/
      health.go

  mocks/
    mock_health_repository.go
    mock_health_usecase.go

pkg/
  httpclient/
  log/
  response/
  validator/
```

## Layer Responsibilities

### `cmd/server`

Application entry point.

Responsibilities:

- Call `container.Run()`.
- Avoid detailed wiring, business logic, or manual configuration.

Rationale:

- `main.go` stays thin, so the application can be tested through `internal/container`.

### `internal/container`

Application composition root.

Responsibilities:

- Load configuration.
- Create the logger.
- Register dependencies in `samber/do`.
- Wire contracts to concrete implementations.
- Create the router and HTTP server.
- Start the server and manage shutdown signals.

Dependency injection is intentionally concentrated in this layer so other packages do not depend on `samber/do`.

Example dependency wiring:

```text
*database.Postgres
    ↓
*infrastructure/repository.HealthRepository
    ↓ as internal/repository.HealthRepository
usecase.HealthUsecase
    ↓
*delivery/http.HealthHandler
    ↓
*delivery/http.Server
```

Infrastructure dependencies such as `*cache.Redis` are also registered here so their health checks and shutdown hooks are managed consistently by the container.

### `internal/domain`

Contains pure domain models.

Current model:

- `HealthStatus`

Rules:

- No JSON tags.
- No knowledge of HTTP response shape.
- No dependency on repository implementations, database/cache drivers, config, logger, or framework code.

Rationale:

- Domain models must be usable by usecases without bringing transport or infrastructure concerns with them.

### `internal/usecase`

Contains application/business usecases.

Current usecase:

- `HealthUsecase`
- `GetHealth(ctx)`

Responsibilities:

- Orchestrate repository contracts.
- Produce domain objects.
- Define application semantics, for example the health endpoint returns application status `ok` even when the database is down.

Rules:

- Depends on `internal/domain` and contracts in `internal/repository`.
- Does not depend on repository implementations.
- Does not know about HTTP, JSON, chi, pgx, Redis, or `samber/do`.

### `internal/repository`

Contains repository contracts required by usecases.

Current contract:

```go
type HealthRepository interface {
    Ping(ctx context.Context) error
}
```

Why contracts live in `internal/repository` instead of infrastructure:

- Usecases define the data-access behavior they need as interfaces.
- Infrastructure only implements those contracts.
- This keeps dependency direction pointing inward toward the application core.

### `internal/infrastructure`

Contains technical details that can be replaced without changing usecases.

#### `infrastructure/config`

Responsibilities:

- Load configuration from defaults, config file, and environment variables.
- Precedence: environment variables > `config.yaml` > defaults.

Configuration includes:

- application metadata,
- port,
- database URL and pool options,
- Redis URL and client options,
- log level/format,
- rate limit,
- HTTP timeouts.

#### `infrastructure/database`

PostgreSQL database wrapper based on `pgxpool`.

Responsibilities:

- Create a pool from a DSN.
- Expose `Ping`, `HealthCheck`, and `Shutdown`.
- Provide `NewWithPool` for tests with a mock pool.

This package uses a narrow interface:

```go
type Pool interface {
    Ping(ctx context.Context) error
    Close()
}
```

Rationale:

- The rest of the code does not depend directly on concrete `*pgxpool.Pool`.
- Tests can use `pgxmock` without a real database.

#### `infrastructure/cache`

Redis client wrapper based on `go-redis`.

Responsibilities:

- Create a Redis client from a Redis URL.
- Apply retry, pool, and timeout options.
- Expose `Ping`, `HealthCheck`, and `Shutdown`.
- Provide `NewWithClient` for tests with a fake or test client.

This package uses a narrow interface:

```go
type Client interface {
    Ping(ctx context.Context) *redis.StatusCmd
    Close() error
}
```

Rationale:

- Redis-specific details stay inside infrastructure.
- Tests can use `miniredis` instead of a real Redis server.
- The Redis client can be replaced or reconfigured without changing core usecases.

#### `infrastructure/repository`

Concrete implementations of repository contracts.

Current implementation:

- `HealthRepository` uses `DBPinger` to check database connectivity.

Design choice:

- Infrastructure repositories receive small interfaces such as `DBPinger`, making both tests and production wiring straightforward.

### `internal/delivery/http`

HTTP delivery layer.

Responsibilities:

- HTTP handlers.
- Request/response DTOs.
- chi router.
- Middleware.
- HTTP server wrapper.

Rules:

- Handlers may depend on usecase interfaces.
- Handlers are responsible for converting domain objects into response DTOs.
- Domain objects do not get JSON tags.

#### Handler

`HealthHandler` calls `HealthUsecase` and converts `domain.HealthStatus` into JSON:

```json
{
  "success": true,
  "data": {
    "status": "ok",
    "database": "up",
    "redis": "up"
  }
}
```

If the database or Redis is down, the response uses HTTP 503 with status `degraded`:

```json
{
  "success": true,
  "data": {
    "status": "degraded",
    "database": "down",
    "redis": "up"
  }
}
```

Rationale:

- The health endpoint reports application status and each dependency status separately.
- A dependency outage makes the service unavailable (503) so load balancers / orchestrators can react, while each dependency's status is still reported individually.

#### Router

The router uses:

- request ID middleware,
- structured request logging,
- panic recoverer,
- timeout middleware,
- optional rate-limit middleware.

Rate limiting is enabled through config:

```yaml
rate_limit:
  rate: 10
  burst: 20
```

If `rate_limit.rate <= 0`, the router does not install the rate-limit middleware.

#### Rate Limit Middleware

The rate limiter uses `golang.org/x/time/rate` with a default key based on the client's `RemoteAddr`.

Design choice:

- The default uses `RemoteAddr` so the application does not blindly trust spoofable proxy headers.
- If the app is deployed behind a reverse proxy, a custom key extractor can be installed through the `WithKeyExtractor` option.

Client limiter cleanup is performed lazily when requests arrive instead of through a permanent background goroutine.

Rationale:

- Simpler lifecycle.
- No cleanup goroutine that lives forever.
- No risk of multiple cleanup goroutines if the handler chain is rebuilt.

### `pkg`

Reusable packages that are not specific to a single feature.

#### `pkg/log`

Thin wrapper over `log/slog`.

Why it exists:

- Provides a small interface for dependency injection.
- Keeps log source locations accurate when wrapper methods are used.
- Provides level/format parsing from config.

#### `pkg/response`

Standard REST response envelope.

Success shape:

```json
{
  "success": true,
  "data": {}
}
```

Error shape:

```json
{
  "success": false,
  "message": "invalid input",
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "invalid input",
    "details": {}
  }
}
```

`WriteJSON` encodes the payload into a buffer before writing the HTTP status so encoding failures can still return a correct HTTP 500 response.

#### `pkg/httpclient`

Wrapper over `gojek/heimdall`.

Responsibilities:

- Provide an HTTP client interface.
- Support timeouts.
- Support retry count and backoff.
- Allow custom HTTP clients for tests or special integrations.

Rationale:

- Consumers do not depend directly on heimdall.
- The retry/backoff library can be replaced without changing callers.

#### `pkg/validator`

Wrapper over `go-playground/validator`.

Responsibilities:

- Validate structs.
- Convert validation errors into a `field -> message` map.
- Use field names from tags, defaulting to `json`.

Rationale:

- Handlers can return `response.ValidationError` with a consistent format.
- Validator library details do not leak into every handler.

## Dependency Rules

Allowed dependency examples:

```text
usecase -> domain
usecase -> repository contract
infrastructure/repository -> repository contract
infrastructure/repository -> infrastructure/database abstraction
http handler -> usecase interface
container -> all layers for wiring
```

Forbidden dependency examples:

```text
domain -> delivery/http
domain -> infrastructure/database
domain -> infrastructure/cache
usecase -> infrastructure/repository
usecase -> pgx
usecase -> redis
repository contract -> infrastructure/database
repository contract -> infrastructure/cache
pkg/response -> internal/usecase
```

## Lifecycle

Startup flow:

```text
main()
  → container.Run()
    → config.Load()
    → log.New(...)
    → container.New(cfg, logger)   // constructs Postgres and Redis eagerly
    → invoke *delivery/http.Server
    → server.ListenAndServe()
    → wait for SIGTERM / interrupt
    → injector.ShutdownOnSignals(...)
```

Shutdown behavior:

- Services registered in `samber/do` can expose shutdown hooks through their `Shutdown()` method.
- `database.Postgres.Shutdown()` closes the database pool.
- `cache.Redis.Shutdown()` closes the Redis client.
- `delivery/http.Server.Shutdown()` gracefully shuts down the HTTP server.

Health check behavior:

- `database.Postgres.HealthCheck(ctx)` delegates to `Ping(ctx)`.
- `cache.Redis.HealthCheck(ctx)` delegates to `Ping(ctx)`.
- Startup diagnostics can run `injector.HealthCheckWithContext(ctx)` to log service health.

## Testing Strategy

Testing follows layer boundaries.

```text
internal/usecase
  test usecases with mocked repository contracts

internal/delivery/http
  test handlers with mocked usecases
  test router/server with httptest or local listener

internal/infrastructure/repository
  test repositories with pgxmock or narrow interface fakes

internal/infrastructure/database
  test pool wrapper behavior with pgxmock and config option application

internal/infrastructure/cache
  test Redis wrapper behavior with miniredis and config option application

internal/container
  smoke test dependency wiring

pkg/*
  unit test reusable package behavior
```

The current coverage target is practical rather than absolute. Generated mocks and the thin `cmd/server/main.go` naturally have low or zero coverage.

## Configuration

Configuration is loaded by `internal/infrastructure/config` with this precedence:

```text
environment variables > config.yaml > defaults
```

Environment variables use the `APP_` prefix, for example:

```text
APP_PORT=:8080
APP_DATABASE_URL=postgres://postgres:postgres@localhost:5432/sbterm?sslmode=disable
APP_REDIS_URL=redis://localhost:6379/0
APP_LOG_LEVEL=info
APP_RATE_LIMIT_RATE=10
APP_RATE_LIMIT_BURST=20
```

Example YAML:

```yaml
port: ":8080"

database:
  url: postgres://postgres:postgres@localhost:5432/sbterm?sslmode=disable

redis:
  url: redis://localhost:6379/0
  max_retries: 3
  pool_size: 10
  min_idle_conns: 0
  dial_timeout: 5s
  read_timeout: 3s
  write_timeout: 3s

rate_limit:
  rate: 10
  burst: 20
```

## Key Architecture Decisions

### Use Clean Architecture Layers

Decision:

- Split code into domain, usecase, repository contract, infrastructure implementation, and delivery.

Why:

- Keeps business/application logic independent from HTTP, database, and cache details.
- Makes testing possible without real infrastructure.
- Keeps the base ready for additional features without mixing concerns.

Trade-off:

- More files and interfaces than a minimal Go server.
- For a single `/health` endpoint this looks verbose, but it provides structure for growth.

### Keep DI in `internal/container`

Decision:

- `samber/do` is used only in the composition root.

Why:

- Prevents the DI framework from leaking into core layers.
- Keeps usecase and repository constructors plain Go.
- Makes packages easier to test directly.

Trade-off:

- The container owns more wiring code.
- Adding a new feature requires updating the container.

### Use Repository Contracts in the Core Layer

Decision:

- Define repository interfaces in `internal/repository` and implementations in `internal/infrastructure/repository`.

Why:

- Usecases depend on the behavior they need, not storage details.
- Infrastructure can change without modifying usecase logic.

Trade-off:

- Small pass-through interfaces may feel repetitive for simple features.

### Use DTOs in the Delivery Layer

Decision:

- HTTP response structs live in `internal/delivery/http`, not in `domain`.

Why:

- Domain remains transport-agnostic.
- JSON tags and API response shape can evolve independently from domain objects.

Trade-off:

- Requires explicit mapping between domain and response DTOs.

### Wrap Common Infrastructure in `pkg`

Decision:

- Logging, response envelope, HTTP client, and validation are wrapped in reusable packages.

Why:

- Keeps behavior consistent across future handlers/features.
- Avoids scattering direct dependency usage across the app.

Trade-off:

- Wrappers should stay thin. If a wrapper mostly duplicates the upstream library without adding policy, reconsider it.

## Adding a New Feature

Recommended flow for a new resource, for example `User`:

1. Add the domain model in `internal/domain/user.go`.
2. Add the repository contract in `internal/repository/user.go`.
3. Add the usecase interface and implementation in `internal/usecase/user.go`.
4. Add the infrastructure implementation in `internal/infrastructure/repository/user.go`.
5. Add HTTP DTOs and handler in `internal/delivery/http/user.go`.
6. Register routes in `internal/delivery/http/router.go`.
7. Wire dependencies in `internal/container/container.go`.
8. Generate mocks for new contracts.
9. Add tests per layer.

Dependency direction should remain:

```text
HTTP handler -> usecase -> repository contract <- infrastructure repository
```

## Managing Transactions (ACID)

To execute database operations within an ACID transaction, `sbterm-server` provides the `TxManager` interface in the `repository` layer. This ensures that operations are safely committed on success and rolled back on error or panic.

### 1. The `TxManager` and `Querier` Pattern

Usecases should depend on `repository.TxManager` to start transactions. Repositories should accept a `repository.Querier` instead of a concrete database pool. `Querier` is an interface satisfied by both the connection pool (for non-transactional queries) and the transaction object (for transactional queries).

### 2. Example: Using Transactions in a Usecase

When a usecase needs to perform multiple repository actions atomically, it calls `WithTx`:

```go
type transferUsecase struct {
    txMgr       repository.TxManager
    accountRepo repository.AccountRepository
}

func (u *transferUsecase) Transfer(ctx context.Context, from, to int64, amount int64) error {
    // Start the transaction
    return u.txMgr.WithTx(ctx, func(tx repository.Querier) error {
        // Pass the `tx` (which implements Querier) to repository methods
        if err := u.accountRepo.Debit(ctx, tx, from, amount); err != nil {
            return err // Returning an error automatically rolls back the transaction
        }
        if err := u.accountRepo.Credit(ctx, tx, to, amount); err != nil {
            return err
        }
        return nil // Returning nil commits the transaction
    })
}
```

### 3. Custom Isolation Levels

If you need a specific isolation level (e.g., `SERIALIZABLE` for strict financial operations), use `WithTxOptions`:

```go
func (u *transferUsecase) StrictTransfer(ctx context.Context, from, to int64, amount int64) error {
    opts := pgx.TxOptions{IsoLevel: pgx.Serializable}
    return u.txMgr.WithTxOptions(ctx, opts, func(tx repository.Querier) error {
        // ...
        return nil
    })
}
```

### 4. Repository Implementation

Repository methods should accept `repository.Querier` as an argument to support running both inside and outside of a transaction. If a query doesn't need to be part of a transaction, you can pass the regular database pool to it:

```go
func (r *accountRepository) Debit(ctx context.Context, db repository.Querier, id int64, amount int64) error {
    _, err := db.Exec(ctx, "UPDATE accounts SET balance = balance - $1 WHERE id = $2", amount, id)
    return err
}
```

## Known Caveats

### Rate limiting behind proxies

Default rate limiting keys by `RemoteAddr`. If deployed behind a reverse proxy, configure a trusted proxy strategy before using forwarded headers as identity.

### `cmd/server` coverage

`cmd/server/main.go` is intentionally thin. It is acceptable for it to have little or no direct test coverage as long as `internal/container` is tested.

### Generated mocks

Generated files in `internal/mocks` are not expected to have direct test coverage.

## Maintenance Guidelines

- Keep `domain` free from transport and infrastructure tags.
- Keep `samber/do` out of usecases, domain, repository contracts, and handlers unless there is a strong reason.
- Prefer narrow interfaces at boundaries.
- Add tests at the layer where behavior lives.
- Document significant architectural changes with ADRs under `doc/adr/` if the decision is expensive to reverse.
