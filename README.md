# sbterm-server

Go server boilerplate with strict Clean Architecture. The layering follows domain → usecase → repository (contract) → infrastructure, with dependency injection via [samber/do](https://github.com/samber/do).

## Tech Stack

- Router: [chi v5](https://github.com/go-chi/chi) + [slog-chi](https://github.com/samber/slog-chi)
- Database: [pgx v5](https://github.com/jackc/pgx) (native `pgxpool`)
- Cache: [go-redis v9](https://github.com/redis/go-redis) with [miniredis](https://github.com/alicebob/miniredis) for tests
- DI: [samber/do v2](https://github.com/samber/do)
- Mocking: [uber-go/mock](https://github.com/uber-go/mock) + [pgxmock](https://github.com/pashagolub/pgxmock)
- Testing: [testify](https://github.com/stretchr/testify)
- Config: [viper](https://github.com/spf13/viper)
- Input validation: [go-playground/validator](https://github.com/go-playground/validator) via `pkg/validator`
- HTTP client: [heimdall v8](https://github.com/gojek/heimdall) via `pkg/httpclient`
- Logging: stdlib `log/slog` via `pkg/log`

## Structure

```text
cmd/server/main.go              # entry point → container.Run()
internal/
  container/                    # DI wiring (the only place using samber/do)
  domain/                       # pure data objects, WITHOUT JSON tags
  delivery/http/                # handlers + DTOs (JSON tags) + chi router + Server
  usecase/                      # interfaces + implementations
  repository/                   # repository contracts used by usecases
  infrastructure/
    cache/                      # Redis wrapper (*Redis, HealthCheck/Shutdown)
    config/                     # config loading from env/config.yaml/defaults
    database/                   # pgxpool wrapper (*Postgres, HealthCheck/Shutdown)
    repository/                 # concrete repository implementations
  mocks/                        # mockgen output (uber-go/mock)
pkg/
  log/                          # slog wrapper (Logger interface + options)
  httpclient/                   # heimdall wrapper (Client interface + options)
  response/                     # REST response envelope + validation errors
  validator/                    # go-playground/validator wrapper (field → message map)
Makefile
.env.example
```

## Commands

```text
make run          # go run ./cmd/server
make build        # go build -o bin/sbterm-server ./cmd/server
make test         # go test ./...
make test-race    # go test -race ./...
make vet          # go vet ./...
make mock         # go generate ./... (regenerate mocks into internal/mocks/)
make tidy         # go mod tidy
```

## Config

Configuration is loaded through [viper](https://github.com/spf13/viper) with this precedence: **env > `config.yaml` > defaults**.

- App metadata: `app.name`, `app.version` (default `dev`, overridable through `APP_NAME`/`APP_VERSION` or `config.yaml`).
- Build-time version can be set through ldflags:
  ```text
  go build -ldflags "-X github.com/nofendian17/sbterm-server/internal/infrastructure/config.version=<tag>" ./cmd/server
  ```
- Environment variables use the `APP_` prefix, for example `APP_PORT`, `APP_DATABASE_URL`, `APP_REDIS_URL`, `APP_RATE_LIMIT_RATE`, and `APP_RATE_LIMIT_BURST`. See `.env.example` for the full list.
- Optional `config.yaml` can be placed at the repository root. It uses dotted/nested keys such as `port`, `database.url`, `redis.url`, `db.max_conns`, `log.level`, `rate_limit.rate`, and `http.read_timeout`. See `config.yaml.example`.

The server can still run when the database is unreachable — the health endpoint reports `database: down` with HTTP 200.

## Middleware

The global stack in `internal/delivery/http/router.go` is: `RequestID` → `slog-chi` (structured logging) → `Recoverer` → `Timeout(30s)` → **RateLimit** → routes. Rate limiting is per-client (token bucket via `golang.org/x/time/rate`) and configurable through `rate_limit.rate`/`APP_RATE_LIMIT_RATE` and `rate_limit.burst`/`APP_RATE_LIMIT_BURST`. Exceeding the limit returns `429`, a `Retry-After` header, and a `TOO_MANY_REQUESTS` response envelope.

Custom middleware lives in `internal/delivery/http/middleware/` and follows the option pattern with table-driven tests.

## Input Validation

`pkg/validator` wraps [go-playground/validator](https://github.com/go-playground/validator). Validation failures become `*ValidationError` with a `map[field]message` using field names from `json` tags, ready to be sent through `response.ValidationError`:

```go
if err := v.Validate(body); err != nil {
    if verr, ok := validator.AsValidationError(err); ok {
        response.ValidationError(w, "validation failed", verr.Fields)
        return
    }
    // other errors, for example invalid input type
}
```

## Tests

Every file with behavior should have table-driven tests using [testify](https://github.com/stretchr/testify) (`assert`/`require`). Current examples:

- `internal/usecase`: gomock `MockHealthRepository`
- `internal/infrastructure/repository`: pgxmock (`ExpectPing`)
- `internal/infrastructure/database`: pgxmock for PostgreSQL wrapper behavior
- `internal/infrastructure/cache`: miniredis for Redis wrapper behavior
- `internal/delivery/http`: httptest + gomock `MockHealthUsecase`
- `internal/container`: DI wiring smoke tests (dead DSN → health 200)
- `pkg/response`: response envelope + validation error behavior
