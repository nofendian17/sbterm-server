# sbterm-server

Go monorepo (single `go.work` workspace, six modules) for a Stockbit real-time price pipeline. Data flows **Stockbit WS → apps/ws → Redpanda/Kafka → apps/ingest → QuestDB**, alongside a REST API (`apps/api`) built on strict Clean Architecture (domain → usecase → repository (contract) → infrastructure) with dependency injection via [samber/do](https://github.com/samber/do).

## Tech Stack

- Router: [chi v5](https://github.com/go-chi/chi) + [slog-chi](https://github.com/samber/slog-chi)
- Messaging: [twmb/franz-go](https://github.com/twmb/franz-go) (Kafka client) against a [Redpanda](https://redpanda.com/) broker
- Stream database: [QuestDB](https://questdb.com/) via [go-questdb-client](https://github.com/questdb/go-questdb-client)
- Database: [pgx v5](https://github.com/jackc/pgx) (native `pgxpool`)
- Cache: [go-redis v9](https://github.com/redis/go-redis) with [miniredis](https://github.com/alicebob/miniredis) for tests
- WebSocket: [gorilla/websocket](https://github.com/gorilla/websocket)
- Protobuf: `libs/proto` (datav1/datafeedv1, Google + platform + securities + social types)
- DI: [samber/do v2](https://github.com/samber/do)
- Mocking: [uber-go/mock](https://github.com/uber-go/mock) + [pgxmock](https://github.com/pashagolub/pgxmock)
- Testing: [testify](https://github.com/stretchr/testify)
- Config: [viper](https://github.com/spf13/viper)
- Input validation: [go-playground/validator](https://github.com/go-playground/validator) via `libs/pkg/validator`
- HTTP client: [heimdall v8](https://github.com/gojek/heimdall) via `libs/pkg/httpclient`
- Logging: stdlib `log/slog` via `libs/pkg/log`

## Structure

```text
apps/
  api/                          # REST API (clean architecture: domain → usecase → repository → infrastructure)
    cmd/server/main.go
    internal/
      container/                # DI wiring (the only place using samber/do in api)
      delivery/http/            # handlers + DTOs (JSON tags) + chi router + Server
      usecase/                  # interfaces + implementations
      repository/               # repository contracts used by usecases
      infrastructure/           # cache, config, database, repository implementations
      mocks/                    # mockgen output (uber-go/mock)
  ws/                           # datafeed subscriber → publishes to Redpanda/Kafka
    cmd/ws/main.go
    internal/
      service/                  # subscription worker + frame router + publisher port
      infrastructure/           # stockbit ws client, kafka producer, config
  ingest/                       # Kafka consumer → writes to QuestDB
    cmd/ingest/main.go
    internal/
      infrastructure/           # questdb + kafka consumer
      service/                  # consume + insert pipeline
libs/
  pkg/                          # shared: log, httpclient, response, validator
  proto/                        # generated/wire-compat protobuf types (datav1, datafeedv1, ...)
  stockbit/                     # stockbit SDK types
Makefile
docker-compose.yml              # api/ws/ingest + postgres, redis, redpanda, questdb
config.api.yaml.example          # apps/api config
config.ws.yaml.example          # apps/ws config
config.ingest.yaml.example      # apps/ingest config
```

## Commands

```text
make run-api       # go run ./apps/api/cmd/server
make run-ws        # go run ./apps/ws/cmd/ws
make run-ingest    # go run ./apps/ingest/cmd/ingest
make build         # build sbterm-api/sbterm-ws/sbterm-ingest into bin/
make test          # go test ./... in every module (apps/api apps/ws apps/ingest libs/pkg libs/proto libs/stockbit)
make test-race     # same, with the race detector
make vet           # go vet ./... in every module
make mock          # cd apps/api && go generate ./... (regenerate mocks)
make tidy          # go work sync then go mod tidy in every module
make fmt           # gofmt -w all Go files under apps/ and libs/
make fmt-check     # fail if any Go file is not gofmt-formatted
make install-hooks # install git hooks (core.hooksPath -> .githooks)
```

`go test ./...`, `go vet ./...` etc. do not work from the workspace root on go1.26; workspace-wide targets iterate the six modules instead.

## Config

Each service reads its own YAML via [viper](https://github.com/spf13/viper): copy `config.api.yaml.example` (api), `config.ws.yaml.example` (ws), or `config.ingest.yaml.example` (ingest) to `config.api.yaml`, `config.ws.yaml`, `config.ingest.yaml` respectively — the live files are gitignored. Keys resolve dotted/nested names like `app.name`, `port`, `stockbit.ws_url`, `redis.url`, `kafka.brokers`, `questdb.url`, `log.level`, and `http.read_timeout`; see the `.example` files for the full schema and defaults.

- App metadata: `app.name`, `app.version` (default `dev`).
- Build-time version can be set through ldflags:
  ```text
  go build -ldflags "-X github.com/nofendian17/sbterm/apps/api/internal/infrastructure/config.version=<tag>" ./apps/api/cmd/server
  ```

The api can still run when a dependency is unreachable — its health endpoint reports `database: down` / `redis: down` with status `degraded` and HTTP 503.

## Middleware

The global stack in `apps/api/internal/delivery/http/router.go` is: `RequestID` → `slog-chi` (structured logging) → `Recoverer` → `Timeout(30s)` → **RateLimit** → routes. Rate limiting is per-client (token bucket via `golang.org/x/time/rate`) and configurable through `rate_limit.rate` and `rate_limit.burst`. Exceeding the limit returns `429`, a `Retry-After` header, and a `TOO_MANY_REQUESTS` response envelope.

Custom middleware lives in `apps/api/internal/delivery/http/middleware/` and follows the option pattern with table-driven tests.

## Input Validation

`libs/pkg/validator` wraps [go-playground/validator](https://github.com/go-playground/validator). Validation failures become `*ValidationError` with a `map[field]message` using field names from `json` tags, ready to be sent through `response.ValidationError`:

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

- `apps/api/internal/usecase`: gomock `MockHealthRepository`
- `apps/api/internal/infrastructure/repository`: pgxmock (`ExpectPing`)
- `apps/api/internal/infrastructure/database`: pgxmock for PostgreSQL wrapper behavior
- `apps/api/internal/infrastructure/cache`: miniredis for Redis wrapper behavior
- `apps/api/internal/delivery/http`: httptest + gomock `MockHealthUsecase`
- `apps/api/internal/container`: DI wiring smoke tests (dead DSN → health 503 degraded)
- `libs/pkg/response`: response envelope + validation error behavior
