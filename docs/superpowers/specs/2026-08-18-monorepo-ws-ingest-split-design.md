# Design: Monorepo restructure + split datafeed ingestion from websocket

**Date:** 2026-08-18
**Status:** Approved

## 1. Context & Problem

`sbterm-server` is a single-module, single-binary Go service mixing three concerns:

1. A ~30-endpoint REST API (Clean Architecture: domain → usecase → repository → infrastructure, DI via samber/do)
2. A Stockbit datafeed **websocket client** (`internal/delivery/ws`) that subscribes per channel config
3. **Data ingestion**: the ws frame handler writes directly into QuestDB sinks (`ws.go`) — transport and persistence are coupled in one callback

The goal is a **go.work monorepo with one module per service**, and to **split data ingestion from the websocket delivery layer** by routing frames through a message broker so the ws consumer and the QuestDB ingester become independent, independently deployable services.

## 2. Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Monorepo shape | `go.work` workspace, one module per deployable | Per-service isolation |
| Service topology | 3 apps + 3 shared libs | api (REST), ws (websocket → broker), ingest (broker → QuestDB) |
| Broker | Redpanda (Kafka-compatible) | Single-node, no ZooKeeper |
| Kafka client | `twmb/franz-go` | Modern, performant |
| Stockbit lib split | Transport-split: REST+auth in `libs/stockbit`, wsclient in `apps/ws` | api never pulls WS code, ws never pulls 40 REST endpoints |
| Delivery semantics | At-least-once | Safe: QuestDB dedup keys collapse replays |

## 3. Target Layout

```
sbterm/
├── go.work                        # lists all 6 modules
├── apps/
│   ├── api/                       module .../sbterm/apps/api      → binary sbterm-api
│   │   ├── cmd/server/main.go
│   │   ├── internal/container/    # DI wiring (no ws/questdb providers)
│   │   ├── internal/delivery/http/**, domain/**, usecase/**, repository/**
│   │   ├── internal/infrastructure/{cache,config,database,repository}/**
│   │   └── internal/mocks/**, tools.go
│   ├── ws/                        module .../sbterm/apps/ws       → binary sbterm-ws
│   │   ├── cmd/main.go
│   │   ├── internal/container/    # refresher + subscriptions + producer
│   │   ├── internal/delivery/ws/  # rewritten: publish frames to Kafka
│   │   ├── internal/infrastructure/stockbit/  # wsclient.go + tests + wire_compat_test.go
│   │   ├── internal/infrastructure/kafka/     # franz-go producer
│   │   └── internal/infrastructure/config/
│   └── ingest/                    module .../sbterm/apps/ingest   → binary sbterm-ingest
│       ├── cmd/main.go
│       ├── internal/container/
│       ├── internal/service/ingest.go          # worker pool → QuestDB sinks
│       ├── internal/infrastructure/questdb/**  # moved as-is
│       ├── internal/infrastructure/kafka/      # franz-go consumer
│       └── internal/infrastructure/config/
└── libs/
    ├── pkg/                       module .../sbterm/libs/pkg      (log, httpclient, response, validator)
    ├── proto/                     module .../sbterm/libs/proto    (generated protobuf)
    └── stockbit/                  module .../sbterm/libs/stockbit (Client REST + auth/token/refresher + GetWebSocketKey)
```

**Dependency graph (acyclic):**

- `libs/pkg` — standalone
- `libs/proto` — standalone
- `libs/stockbit` → `libs/pkg`
- `apps/api` → `libs/pkg`, `libs/stockbit`
- `apps/ws` → `libs/pkg`, `libs/proto`, `libs/stockbit`
- `apps/ingest` → `libs/pkg`, `libs/proto`

Module paths use the repo-root prefix `github.com/nofendian17/sbterm/…` (independent of local folder name).

## 4. Module Contents & Moves (git mv, no behavior change)

- **libs/pkg** ← `pkg/{log,httpclient,response,validator}` + tests
- **libs/proto** ← `internal/infrastructure/stockbit/proto/**`. Rewrite the module prefix in every `.pb.go` import line (mechanical). Embedded descriptor GoPackage strings (the `B…Z…` raw bytes) go stale but are harmless: proto registry keys are `.proto` filenames, which are unchanged — wire compat tests still pass.
- **libs/stockbit** ← `client.go`, `auth.go`, `token.go`, `refresher.go`, `websocket.go`, `profile.go`, all REST endpoint files + tests. Exclusive of `wsclient.go`.
- **apps/api** ← REST service code; `internal/container` loses the QuestDB provider and `deliveryws.Service` provider; `cmd/server/main.go` loses the ws startup block.
- **apps/ws** ← `wsclient.go` + tests + `wire_compat_test.go`, new `cmd`, new `container`, rewritten `delivery/ws`, ws config (drop REST/repo/trade code), kafka producer.
- **apps/ingest** ← `questdb/**` + test, new `cmd`, `container`, ingestion worker pool, ingest config, kafka consumer.

## 5. Kafka Design (franz-go)

- **Topics** (configurable default): `datafeed.running_trade_batch`, `datafeed.order_book`
- **Wire format:** value = serialized protobuf of the channel message (`RunningTradeBatch` / `Orderbook`); key = symbol, so one symbol's order-book frames stay ordered in a single partition
- **Producer (ws):** one `kgo.Client`. The frame handler in `delivery/ws.run` is rewritten from QuestDB-sink writes to routing:
  - `msg.GetRunningTradeBatch()` → produce `RunningTradeBatch`, key = first trade's symbol
  - `msg.GetOrderbook()` → produce `Orderbook`, key = `GetStockCode()`
  - ws keeps zero QuestDB dependency
- **Consumer (ingest):** one `kgo.Client`, consumer group `sbterm-ingest`, `ConsumeTopics(both)`. `PollFetches` loop → worker pool; each worker owns one QuestDB sink per type (sinks remain per-goroutine as today)
- **Failure handling:**
  - Producer errors → log and drop; ws's reconnect loop is the backpressure
  - Consumer parse/store errors → log and continue; uncommitted offsets redeliver, collapsed by QuestDB dedup

## 6. Config

- `config.yaml` (api): `app`, `port`, `database`, `redis`, `stockbit` (base_url/timeout/retry/creds), `log`, `rate_limit`, `http` — drops `questdb` and `ws_*`
- `config.ws.yaml`: `stockbit` (ws_url, ws_subscriptions, ws_ping_interval, ws_reconnect_backoff_*, player creds for refresher key+token), `redis` (token store), `kafka` (brokers, topics, flush interval), `log`
- `config.ingest.yaml`: `questdb` (url/table/order_book_table), `kafka` (brokers, group, topics, workers), `log`
- Viper defaults per module, same `Load()` pattern; ldflags version path becomes `<module>/internal/infrastructure/config.version`

## 7. Packaging & Deploy

- **Dockerfiles:** one per app (`apps/*/Dockerfile`); build from repo root via go.work (`go build ./apps/<app>/cmd/server`); per-app ldflags
- **docker-compose:** add `redpanda` (single node, `rpk cluster health` check); services:
  - `api` depends_on postgres, redis
  - `ws` depends_on redpanda, redis
  - `ingest` depends_on redpanda, questdb
- **Makefile:** workspace-wide `test / test-race / vet / fmt / fmt-check / mock / tidy` + `run-api / run-ws / run-ingest` + `build`
- **CI** (`.github/workflows/test.yml`): `go work sync` then `go test -race ./...` at root; gofmt check unchanged; pre-commit hook unchanged
- **README**: structure + commands update; config doc split

## 8. Testing

- **Moved as-is:** libs/pkg, libs/stockbit tests, all API tests, wsclient + wire compat tests (apps/ws), questdb tests (apps/ingest)
- **New unit tests:**
  - ws: frame router table-driven (frame → topic/key/value) with a fake publisher; container smoke
  - ingest: record handler table-driven (bytes → sink call / error paths); container smoke
- **End-to-end smoke:** `docker compose up`; verify a real datafeed frame lands as QuestDB rows via `select` from the questdb console

## 9. Non-Goals / Deferred

- No HTTP health endpoint for ws/ingest (relies on `restart: always`)
- No exactly-once delivery, no producer retry/queue persistence concerns (dedup covers replays)
- QuestDB schema/design untouched
- Trade query endpoints stay on the API's Stockbit REST repos (no reading back from QuestDB)