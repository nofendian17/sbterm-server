# Monorepo + ws/ingest Split Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Restructure the single-module `sbterm-server` repo into a `go.work` monorepo with one module per service (`api`, `ws`, `ingest`) plus shared libs, and split datafeed ingestion from the websocket delivery layer by routing frames through Redpanda/Kafka via `twmb/franz-go`.

**Architecture:** Three deployable services in one workspace. `apps/ws` runs the Stockbit datafeed websocket clients, decodes protobuf frames, and publishes them to Kafka topics. `apps/ingest` consumes those topics and writes to QuestDB. `apps/api` keeps the existing REST API unchanged in behavior. Shared generic packages live in `libs/pkg`, generated protobuf in `libs/proto`, and the Stockbit REST client + auth/refresher in `libs/stockbit`.

**Tech Stack:** Go 1.26.5, `go.work` workspace, `github.com/twmb/franz-go`, Redpanda, QuestDB (QWP), samber/do DI, viper config, chi REST.

**Spec:** `docs/superpowers/specs/2026-08-18-monorepo-ws-ingest-split-design.md`

## Global Constraints

- Module paths are `github.com/nofendian17/sbterm/{libs|apps}/<name>` — the root `go.mod`/`go.sum` are **deleted** and replaced by `go.work`
- At-least-once Kafka delivery is acceptable: QuestDB dedup keys collapse replays
- Generated `.pb.go` files are **moved, never regenerated** (no `.proto` sources exist in-repo); only their Go import lines are rewritten
- `internal/` visibility rule applies per module; cross-module imports only ever touch public packages in `libs/*`
- No behavior change to the REST API, questdb schema handling, or wsclient reconnect logic
- Every task must leave the module graph compiling and `make test` green before commit

---

## File Structure (post-move target)

```
apps/api/internal/delivery/http/**        # 30 handlers + router + server + middleware (moved)
apps/api/internal/{domain,usecase,repository}/**   # moved
apps/api/internal/infrastructure/{cache,config,database,repository}/**   # moved
apps/api/internal/container/**            # trimmed: no ws/questdb providers
apps/api/internal/mocks/**                # moved
apps/api/cmd/server/main.go               # moved; ws startup block removed
apps/ws/internal/delivery/ws/ws.go        # REWRITTEN: publish frames to Kafka
apps/ws/internal/delivery/ws/router.go    # NEW: frame → topic/key/value
apps/ws/internal/delivery/ws/ws_test.go   # NEW: router unit tests
apps/ws/internal/infrastructure/stockbit/{wsclient.go,wsclient_test.go}   # moved
apps/ws/internal/infrastructure/stockbit/wire_compat_test.go              # moved
apps/ws/internal/infrastructure/kafka/producer.go   # NEW: franz-go producer
apps/ws/internal/container/container.go   # NEW: refresher + subs + producer + ws
apps/ws/internal/container/container_test.go        # NEW: wiring smoke
apps/ws/internal/infrastructure/config/config.go    # NEW: ws-scoped config
apps/ws/cmd/ws/main.go                    # NEW
apps/ingest/internal/service/handler.go   # NEW: record bytes → QuestDB sink
apps/ingest/internal/service/handler_test.go        # NEW: unit tests
apps/ingest/internal/service/ingest.go    # NEW: PollFetches loop → handler
apps/ingest/internal/infrastructure/kafka/consumer.go  # NEW: franz-go consumer
apps/ingest/internal/infrastructure/questdb/**      # moved as-is
apps/ingest/internal/container/container.go         # NEW
apps/ingest/internal/container/container_test.go    # NEW: wiring smoke
apps/ingest/internal/infrastructure/config/config.go  # NEW: ingest-scoped config
apps/ingest/cmd/ingest/main.go            # NEW
libs/pkg/{log,httpclient,response,validator}/**     # moved from pkg/
libs/proto/**                             # moved from internal/infrastructure/stockbit/proto/
libs/stockbit/{client.go,auth.go,token.go,refresher.go,websocket.go,profile.go} + REST endpoints + tests   # moved
go.work                                   # NEW
Makefile                                  # REWRITTEN (workspace-wide)
apps/{api,ws,ingest}/Dockerfile           # NEW
config.yaml / config.ws.yaml / config.ingest.yaml (+ .example)   # split config
docker-compose.yml                        # REWRITTEN: redpanda + 3 apps
.github/workflows/test.yml                # workspace-aware
README.md                                 # updated
.tool-versions / tools.go                 # tools.go moves to apps/api
```

---

## Phase 0 — Workspace scaffold

### Task 1: Create go.work and delete root module

**Files:**
- Create: `go.work`
- Delete: `go.mod`, `go.sum`

**Interfaces:**
- Produces: workspace with module paths `github.com/nofendian17/sbterm/apps/{api,ws,ingest}` and `github.com/nofendian17/sbterm/libs/{pkg,proto,stockbit}`. All later tasks assume these six module dirs exist with a `go.mod` declaring exactly that path.

- [ ] **Step 1: Create the six module directories**

```bash
mkdir -p apps/api apps/ws apps/ingest libs/pkg libs/proto libs/stockbit
```

- [ ] **Step 2: Initialize module seeds (paths must match exactly)**

```bash
go mod init github.com/nofendian17/sbterm/libs/pkg        && : > libs/pkg/go.mod.tmp
go mod init github.com/nofendian17/sbterm/libs/proto      && : > libs/proto/go.mod.tmp
go mod init github.com/nofendian17/sbterm/libs/stockbit   && : > libs/stockbit/go.mod.tmp
go mod init github.com/nofendian17/sbterm/apps/api        && : > apps/api/go.mod.tmp
go mod init github.com/nofendian17/sbterm/apps/ws         && : > apps/ws/go.mod.tmp
go mod init github.com/nofendian17/sbterm/apps/ingest     && : > apps/ingest/go.mod.tmp
rm apps/*/go.mod.tmp libs/*/go.mod.tmp
```

> This is only a seed — each phase runs `go mod tidy` inside the module once its files land.

- [ ] **Step 3: Build go.work**

Write `go.work`:

```go
go 1.26.5

use (
	./apps/api
	./apps/ws
	./apps/ingest
	./libs/pkg
	./libs/proto
	./libs/stockbit
)
```

- [ ] **Step 4: Delete the root module**

```bash
git rm go.mod go.sum
```

- [ ] **Step 5: Commit**

```bash
git add go.work apps libs
git commit -m "chore(monorepo): scaffold go.work workspace with six module slots"
```

---

## Phase 1 — libs/pkg

### Task 2: Move shared generic packages into libs/pkg

**Files:**
- Move `pkg/` → `libs/pkg/`
- Modify: every `.go` file under `libs/pkg` that imports something

**Interfaces:**
- Produces: `github.com/nofendian17/sbterm/libs/pkg/log` (type `log.Logger`, `log.New(opts...)`), `.../libs/pkg/httpclient`, `.../libs/pkg/response`, `.../libs/pkg/validator` — signatures unchanged from the old `github.com/nofendian17/sbterm-server/pkg/*`. All later modules import these under the new path.

- [ ] **Step 1: Move the tree**

```bash
git mv pkg libs/pkg
```

- [ ] **Step 2: Rewrite internal imports**

```bash
grep -rl 'sbterm-server/pkg/' libs/pkg | xargs -r perl -pi -e 's#github\.com/nofendian17/sbterm-server/pkg/#github.com/nofendian17/sbterm/libs/pkg/#g'
```

- [ ] **Step 3: Tidy and verify**

```bash
(cd libs/pkg && go mod tidy && go test ./... && go vet ./...)
```

Expected: all tests pass, no compile errors. Dependencies pulled in: `github.com/gojek/heimdall/v8` (httpclient), `github.com/go-playground/validator/v10` (validator).

- [ ] **Step 4: Commit**

```bash
git add libs/pkg
git commit -m "refactor(monorepo): move pkg to libs/pkg module"
```

---

## Phase 2 — libs/proto

### Task 3: Move generated protobuf into libs/proto

**Files:**
- Move `internal/infrastructure/stockbit/proto/` → `libs/proto/`
- Modify: import lines in all `.pb.go` files

**Interfaces:**
- Produces: `github.com/nofendian17/sbterm/libs/proto/securities/transactional/datafeed/v1` (imported as `datafeedv1`), `.../libs/proto/securities/transactional/datafeed/consumer/entity/v1` (`consumerv1`), plus every other proto package under the old `.../stockbit/proto/` tree mapped to `.../libs/proto/`. Wire compat message names are unchanged.

- [ ] **Step 1: Move the tree**

```bash
git mv internal/infrastructure/stockbit/proto libs/proto
```

- [ ] **Step 2: Rewrite the module prefix in every .pb.go import + embedded file header reference**

The long-prefix replace first (so `.../stockbit/proto/securities/...` is not caught by a later stockbit rule):

```bash
grep -rl 'sbterm-server/internal/infrastructure/stockbit/proto/' libs/proto | xargs -r perl -pi -e 's#github\.com/nofendian17/sbterm-server/internal/infrastructure/stockbit/proto/#github.com/nofendian17/sbterm/libs/proto/#g'
```

> The raw `B…Z…` descriptor strings (GoPackage metadata) inside the generated byte blobs deliberately do **not** change. They are informational only; `proto.RegisterFile` keys off the `.proto` filename, which is unchanged.

- [ ] **Step 3: Tidy and verify**

```bash
(cd libs/proto && go mod tidy && go build ./... && go vet ./...)
```

Expected: builds. Deps: `google.golang.org/protobuf`.

- [ ] **Step 4: Commit**

```bash
git add libs/proto apps libs internal 2>/dev/null; git add -u
git commit -m "refactor(monorepo): move generated protobuf to libs/proto module"
```

---

## Phase 3 — libs/stockbit

### Task 4: Move Stockbit REST client + auth into libs/stockbit

**Files:**
- Move from `internal/infrastructure/stockbit/` → `libs/stockbit/` EXCEPT `wsclient.go`, `wsclient_test.go`, `wire_compat_test.go`:
  - `client.go`, `auth.go`, `token.go`, `refresher.go`, `websocket.go`, `profile.go`
  - all REST endpoint files (`activity.go`, `brokertop.go`, … 40+ files) and their `*_test.go`
  - `client_auth_test.go`, `refresher_test.go`, `client_test.go`
- Modify: import lines referring to `pkg/` and to the proto module

**Interfaces:**
- Produces (all under `github.com/nofendian17/sbterm/libs/stockbit`): type `*stockbit.Client`, `*stockbit.Refresher`, `New(opts...)`, `NewRefresher(...)`, `NewRedisTokenStore(...)`, `WithTimeout/WithRetryCount/WithLogger/WithBaseURL`, `(c *Client) GetWebSocketKey(ctx)`.
- Consumes: `libs/pkg` (`log`, `httpclient`). The `wsclient.go`-only symbols (`WSClient`, `WSHandler`, `WSSubscription`, `MergeWSChannels`, `WSChannel*` builders, `WSSubscription.UserID`) are NOT in this module — they are produced by Task 6 inside `apps/ws/internal/infrastructure/stockbit`.

- [ ] **Step 1: Move the chosen files**

```bash
cd internal/infrastructure/stockbit
git mv client.go auth.go token.go refresher.go websocket.go profile.go ../../libs/stockbit/
git mv client_auth_test.go refresher_test.go client_test.go ../../libs/stockbit/
# each REST endpoint file + its test:
for f in activity activity_historical brokertop chartbit corpaction findata_financial foreigndomestic fundachart fundachart_metrics historicalsummary index indexsummary keystats majorholder marketdetector mover orderbook orderqueue priceperformance runningtrade search sectors session shareholding_composition shareholding_network stream subsidiary topstock trending; do
  ls ${f}*.go 2>/dev/null | grep -v wsclient | xargs -r git mv -t /tmp 2>/dev/null || true
done
```

> Move-rest-of-files fallback: everything `.go` left in `internal/infrastructure/stockbit/` after this step except `wsclient.go`/`wsclient_test.go`/`wire_compat_test.go` goes to `libs/stockbit/`. The leftovers for Task 6 are exactly the three ws files.

- [ ] **Step 2: Rewrite imports in libs/stockbit**

```bash
cd /home/beni/Projects/go/lab/sbterm-server
grep -rl 'sbterm-server/internal/infrastructure/stockbit/proto/' libs/stockbit | xargs -r perl -pi -e 's#github\.com/nofendian17/sbterm-server/internal/infrastructure/stockbit/proto/#github.com/nofendian17/sbterm/libs/proto/#g'
grep -rl 'sbterm-server/pkg/' libs/stockbit | xargs -r perl -pi -e 's#github\.com/nofendian17/sbterm-server/pkg/#github.com/nofendian17/sbterm/libs/pkg/#g'
grep -rl 'sbterm-server/internal/' libs/stockbit | xargs -r perl -pi -e 's#github\.com/nofendian17/sbterm-server/internal/#github.com/nofendian17/sbterm/libs/#g'
```

> The third rule fixes any leftover cross-references (e.g. test helpers importing the old stockbit package path `.../internal/infrastructure/stockbit"` → `.../libs/stockbit"`). Check `grep -rn 'sbterm-server' libs/stockbit` returns nothing after.

- [ ] **Step 3: Tidy and verify**

```bash
(cd libs/stockbit && go mod tidy && go test ./... && go vet ./...)
```

Expected: all stockbit REST/client/auth tests pass. `go.mod` requires `libs/pkg` (via `replace`? No — same workspace; plain `require github.com/nofendian17/sbterm/libs/pkg` is resolved by go.work).

- [ ] **Step 4: Commit**

```bash
git add libs/stockbit
git commit -m "refactor(monorepo): extract libs/stockbit module (REST client, auth, refresher)"
```

---

## Phase 4 — apps/api

### Task 5: Move the REST API into apps/api

**Files:**
- Move: `cmd/server/` → `apps/api/cmd/server/`
- Move: `internal/container/`, `internal/delivery/http/`, `internal/domain/`, `internal/usecase/`, `internal/repository/`, `internal/mocks/`, `internal/infrastructure/{cache,config,database,repository}/`
- Move: `tools.go` → `apps/api/tools.go`
- Leave in place: `internal/infrastructure/questdb/`, `internal/infrastructure/stockbit/` (now empty except ws files)
- Modify: `apps/api/internal/container/container.go` — remove the `questdb` provider and the `deliveryws.Service` provider; drop those imports
- Modify: `apps/api/internal/container/container.go` `Run()` — remove the ws startup block and `deliveryws` import
- Modify: `apps/api/internal/infrastructure/config/config.go` — drop `QuestDBConfig`, `WSURL`, `WSSubscriptions`, `WSPingInterval`, `WSReconnectBackoffInitial`, `WSReconnectBackoffMax` from `StockbitConfig`, and their defaults
- Modify: `apps/api/internal/container/container_test.go` — import `libs/stockbit` instead of `internal/infrastructure/stockbit`

**Interfaces:**
- Consumes: `libs/stockbit` (`*stockbit.Client`, `*stockbit.Refresher`, `NewRedisTokenStore`, options), `libs/pkg` (`log`).
- Produces: `github.com/nofendian17/sbterm/apps/api/internal/container.New(cfg, logger)` and `.Run()`, `.../apps/api/internal/delivery/http.Server`, binary `apps/api/cmd/server`.

- [ ] **Step 1: Move the directories**

```bash
git mv cmd/server apps/api/cmd/server
git mv internal/mocks apps/api/internal/mocks
git mv tools.go apps/api/tools.go
for d in container delivery domain usecase repository; do
  git mv internal/$d apps/api/internal/$d
done
git mv internal/infrastructure/cache apps/api/internal/infrastructure/cache
git mv internal/infrastructure/config apps/api/internal/infrastructure/config
git mv internal/infrastructure/database apps/api/internal/infrastructure/database
git mv internal/infrastructure/repository apps/api/internal/infrastructure/repository
mkdir -p apps/api/internal/infrastructure
```

- [ ] **Step 2: Rewrite imports inside apps/api**

```bash
cd apps/api
grep -rl 'sbterm-server/internal/infrastructure/stockbit/proto/' . | xargs -r perl -pi -e 's#github\.com/nofendian17/sbterm-server/internal/infrastructure/stockbit/proto/#github.com/nofendian17/sbterm/libs/proto/#g'
grep -rl 'sbterm-server/internal/infrastructure/stockbit\b' . | xargs -r perl -pi -e 's#github\.com/nofendian17/sbterm-server/internal/infrastructure/stockbit#github.com/nofendian17/sbterm/libs/stockbit#g'
grep -rl 'sbterm-server/pkg/' . | xargs -r perl -pi -e 's#github\.com/nofendian17/sbterm-server/pkg/#github.com/nofendian17/sbterm/libs/pkg/#g'
grep -rl 'sbterm-server/internal/' . | xargs -r perl -pi -e 's#github\.com/nofendian17/sbterm-server/internal/#github.com/nofendian17/sbterm/apps/api/internal/#g'
```

> Order matters: proto and stockbit and pkg patterns run BEFORE the generic `internal/` rewrite, otherwise `.../internal/infrastructure/stockbit` becomes mangled. After all four: `grep -rn 'sbterm-server/' .` must return nothing.

- [ ] **Step 3: Trim config.go**

In `apps/api/internal/infrastructure/config/config.go`:
- Delete `QuestDB QuestDBConfig` from `Config`
- Delete `QuestDBConfig` struct
- Delete `WSURL`, `WSSubscriptions`, `WSPingInterval`, `WSReconnectBackoffInitial`, `WSReconnectBackoffMax` fields from `StockbitConfig`
- Delete `WSSubscriptionConfig` and `WSChannelConfig` structs
- Delete these defaults: `questdb.*` (3 lines), `stockbit.ws_url`, `stockbit.ws_ping_interval`, `stockbit.ws_reconnect_backoff_initial`, `stockbit.ws_reconnect_backoff_max`
- Keep `version` var: update its doc comment to `github.com/nofendian17/sbterm/apps/api/internal/infrastructure/config.version`

- [ ] **Step 4: Trim container.go**

In `apps/api/internal/container/container.go`:
- Remove imports: `deliveryws ".../delivery/ws"`, `"github.com/nofendian17/sbterm/apps/api/internal/infrastructure/questdb"` (was `.../sbterm-server/internal/infrastructure/questdb`)
- In `provideInfrastructure`: delete the `*questdb.Client` provider block
- In `provideStockbit`: delete the `*deliveryws.Service` provider block entirely (the `for ... WSSubscriptions` loop). Keep the `*stockbit.Refresher` and `*stockbit.Client` providers intact.
- In `Run()`: delete the `ws_subscriptions` / `wsSvc` block (the `if len(cfg.Stockbit.WSSubscriptions) == 0 {...} else {...}`), replacing it with nothing. Remove `deliveryws` references.

- [ ] **Step 5: Move api config file**

```bash
git mv config.yaml apps/api/config.yaml
git mv config.yaml.example apps/api/config.yaml.example
```

- [ ] **Step 6: Tidy and verify**

```bash
(cd apps/api && go mod tidy && go generate ./... && go test ./... && go vet ./...)
```

Expected: all API tests green (usecase/http/container/repository/infrastructure). `container_test.go` compiles against `libs/stockbit` (its `TestStockbitClientIsAuthenticatedWhenResolvedFirst` still resolves `*stockbit.Client`/`*stockbit.Refresher` from the same names in the moved module).

- [ ] **Step 7: Commit**

```bash
git add apps/api
git commit -m "refactor(monorepo): move REST API into apps/api module"
```

---

## Phase 5 — apps/ws

### Task 6: Move wsclient into apps/ws

**Files:**
- Move: `internal/infrastructure/stockbit/{wsclient.go,wsclient_test.go,wire_compat_test.go}` → `apps/ws/internal/infrastructure/stockbit/`
- Modify: their imports to the new `libs/proto` and `libs/pkg` paths

**Interfaces:**
- Produces: `github.com/nofendian17/sbterm/apps/ws/internal/infrastructure/stockbit` with `NewWSClient(url, key KeyProvider, opts...)`, `WSClient.Run(ctx, WSSubscription, WSHandler)`, `WSHandler` (signature `func(ctx context.Context, m *datafeedv1.WebsocketWrapMessageChannel) error`), `WSChannel*` builders, `MergeWSChannels`, `Refresher` still comes from `libs/stockbit`.

- [ ] **Step 1: Move and rewrite**

```bash
mkdir -p apps/ws/internal/infrastructure/stockbit apps/ws/internal/infrastructure/config apps/ws/internal/infrastructure/kafka apps/ws/internal/delivery/ws apps/ws/internal/container apps/ws/cmd/ws
git mv internal/infrastructure/stockbit/wsclient.go apps/ws/internal/infrastructure/stockbit/
git mv internal/infrastructure/stockbit/wsclient_test.go apps/ws/internal/infrastructure/stockbit/
git mv internal/infrastructure/stockbit/wire_compat_test.go apps/ws/internal/infrastructure/stockbit/
cd apps/ws
grep -rl 'sbterm-server/internal/infrastructure/stockbit/proto/' . | xargs -r perl -pi -e 's#github\.com/nofendian17/sbterm-server/internal/infrastructure/stockbit/proto/#github.com/nofendian17/sbterm/libs/proto/#g'
grep -rl 'sbterm-server/internal/infrastructure/stockbit\b' . | xargs -r perl -pi -e 's#github\.com/nofendian17/sbterm-server/internal/infrastructure/stockbit#github.com/nofendian17/sbterm/libs/stockbit#g'
grep -rl 'sbterm-server/pkg/' . | xargs -r perl -pi -e 's#github\.com/nofendian17/sbterm-server/pkg/#github.com/nofendian17/sbterm/libs/pkg/#g'
grep -rl 'sbterm-server/internal/' . | xargs -r perl -pi -e 's#github\.com/nofendian17/sbterm-server/internal/#github.com/nofendian17/sbterm/apps/ws/internal/#g'
```

- [ ] **Step 2: Tidy and verify the moved tests compile**

```bash
(cd apps/ws && go mod tidy && go test ./internal/infrastructure/stockbit/... && go vet ./internal/infrastructure/stockbit/...)
```

Expected: wsclient + wire compat tests run green. The ws module manages `github.com/twmb/franz-go`, `github.com/gorilla/websocket`, `google.golang.org/protobuf`, `libs/pkg`, `libs/proto`, `libs/stockbit` (rest of deps resolved by tidy).

- [ ] **Step 3: Commit**

```bash
git add apps/ws
git commit -m "refactor(monorepo): move datafeed wsclient into apps/ws module"
```

### Task 7: Kafka publisher wrapper (ws)

**Files:**
- Create: `apps/ws/internal/infrastructure/kafka/producer.go`
- Test: none (thin wrapper; routing logic is covered by Task 8)

**Interfaces:**
- Produces:

```go
package kafka

type Topics struct {
	RunningTradeBatch string
	OrderBook         string
}

type Publisher interface {
	Publish(ctx context.Context, topic string, key string, value []byte) error
	Close()
}

func NewProducer(brokers []string, logger log.Logger) (*Producer, error)
```

- **Step 1: Write `producer.go`**

```go
package kafka

import (
	"context"
	"fmt"

	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/nofendian17/sbterm/libs/pkg/log"
)

// Topics names the Kafka topics used by the datafeed pipeline.
type Topics struct {
	RunningTradeBatch string
	OrderBook         string
}

// Publisher sends one protobuf frame to a topic. Implementations must be safe
// for concurrent use.
type Publisher interface {
	Publish(ctx context.Context, topic string, key string, value []byte) error
	Close()
}

// Producer publishes datafeed frames to Redpanda/Kafka via franz-go.
type Producer struct {
	client *kgo.Client
	logger log.Logger
}

// NewProducer builds a franz-go producer seeded with the given brokers.
func NewProducer(brokers []string, logger log.Logger) (*Producer, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.RequiredAcks(kgo.AllISRAcks()),
		kgo.DefaultProduceTopic(""),
	)
	if err != nil {
		return nil, fmt.Errorf("kafka: new producer: %w", err)
	}
	return &Producer{client: client, logger: logger}, nil
}

// Publish sends one record synchronously. A non-nil error means the record was
// not acknowledged; the ws reconnect loop provides the backpressure.
func (p *Producer) Publish(ctx context.Context, topic string, key string, value []byte) error {
	res := p.client.ProduceSync(ctx, &kgo.Record{Topic: topic, Key: []byte(key), Value: value})
	if err := res.FirstErr(); err != nil {
		return fmt.Errorf("kafka: produce %s: %w", topic, err)
	}
	return nil
}

// Close shuts down the producer and flushes buffered records.
func (p *Producer) Close() {
	p.client.Close()
}
```

- [ ] **Step 2: Verify it compiles**

```bash
(cd apps/ws && go build ./internal/infrastructure/kafka/...)
```

- [ ] **Step 3: Add the franz-go dependency and commit**

```bash
(cd apps/ws && go mod tidy)
git add apps/ws/internal/infrastructure/kafka apps/ws/go.mod apps/ws/go.sum
git commit -m "feat(ws): add franz-go kafka producer wrapper"
```

### Task 8: Rewrite delivery/ws to publish frames (TDD)

**Files:**
- Rewrite: `apps/ws/internal/delivery/ws/ws.go` (was moved in a previous phase? No — it still sits at `internal/delivery/ws/`; move it first)
- Create: `apps/ws/internal/delivery/ws/router.go`
- Test: `apps/ws/internal/delivery/ws/router_test.go`

**Interfaces:**
- Consumes: `kafka.Publisher`, `kafka.Topics`, `libs/stockbit` `Refresher`, `apps/ws/internal/infrastructure/stockbit` `WSClient`.
- Produces:

```go
type Subscription struct {
	Name    string
	Client  *stockbit.WSClient
	Channel *datafeedv1.WebsocketChannel
}

type FrameRouter struct {	publisher kafka.Publisher; topics kafka.Topics }
func NewFrameRouter(publisher kafka.Publisher, topics kafka.Topics) *FrameRouter
func (r *FrameRouter) Route(ctx context.Context, m *datafeedv1.WebsocketWrapMessageChannel) error

type Service struct { ... }
func New(subs []*Subscription, refresher *stockbit.Refresher, router *FrameRouter, logger log.Logger) *Service
func (s *Service) Start()
func (s *Service) Shutdown() error
```

- [ ] **Step 1: Move the old delivery/ws and strip QuestDB**

```bash
git mv internal/delivery/ws apps/ws/internal/delivery/ws
cd apps/ws/internal/delivery/ws
perl -pi -e 's#github\.com/nofendian17/sbterm-server/internal/infrastructure/stockbit/proto/#github.com/nofendian17/sbterm/libs/proto/#g; s#github\.com/nofendian17/sbterm-server/internal/infrastructure/stockbit#github.com/nofendian17/sbterm/libs/stockbit#g; s#github\.com/nofendian17/sbterm-server/pkg/#github.com/nofendian17/sbterm/libs/pkg/#g; s#github\.com/nofendian17/sbterm-server/#github.com/nofendian17/sbterm/apps/ws/#g' ws.go
```

- [ ] **Step 2: Write the failing router test** (`router_test.go`)

```go
package ws

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	kafkapkg "github.com/nofendian17/sbterm/apps/ws/internal/infrastructure/kafka"
	consumerv1 "github.com/nofendian17/sbterm/libs/proto/securities/transactional/datafeed/consumer/entity/v1"
	datafeedv1 "github.com/nofendian17/sbterm/libs/proto/securities/transactional/datafeed/v1"
)

type fakePublisher struct {
	topics []record
	closed bool
}

type record struct {
	topic string
	key   string
	value []byte
}

func (f *fakePublisher) Publish(_ context.Context, topic string, key string, value []byte) error {
	f.topics = append(f.topics, record{topic: topic, key: key, value: append([]byte(nil), value...)})
	return nil
}
func (f *fakePublisher) Close() { f.closed = true }

func TestFrameRouter(t *testing.T) {
	topics := kafkapkg.Topics{RunningTradeBatch: "datafeed.running_trade_batch", OrderBook: "datafeed.order_book"}

	t.Run("running trade batch publishes to its topic keyed by first symbol", func(t *testing.T) {
		pub := &fakePublisher{}
		router := NewFrameRouter(pub, topics)

		batch := &datafeedv1.RunningTradeBatch{Batch: []*datafeedv1.RunningTrade{{Stock: "BBRI", Price: 1000}}}
		msg := &datafeedv1.WebsocketWrapMessageChannel{Message: &datafeedv1.WebsocketWrapMessageChannel_RunningTradeBatch{RunningTradeBatch: batch}}

		require.NoError(t, router.Route(context.Background(), msg))
		require.Len(t, pub.topics, 1)
		assert.Equal(t, "datafeed.running_trade_batch", pub.topics[0].topic)
		assert.Equal(t, "BBRI", pub.topics[0].key)

		got := &datafeedv1.RunningTradeBatch{}
		require.NoError(t, proto.Unmarshal(pub.topics[0].value, got))
		assert.Equal(t, "BBRI", got.GetBatch()[0].GetStock())
	})

	t.Run("order book publishes to its topic keyed by stock code", func(t *testing.T) {
		pub := &fakePublisher{}
		router := NewFrameRouter(pub, topics)

		ob := &consumerv1.Orderbook{StockCode: "BBCA"}
		msg := &datafeedv1.WebsocketWrapMessageChannel{Message: &datafeedv1.WebsocketWrapMessageChannel_Orderbook{Orderbook: ob}}

		require.NoError(t, router.Route(context.Background(), msg))
		require.Len(t, pub.topics, 1)
		assert.Equal(t, "datafeed.order_book", pub.topics[0].topic)
		assert.Equal(t, "BBCA", pub.topics[0].key)

		got := &consumerv1.Orderbook{}
		require.NoError(t, proto.Unmarshal(pub.topics[0].value, got))
		assert.Equal(t, "BBCA", got.GetStockCode())
	})

	t.Run("frames without an ingested channel publish nothing", func(t *testing.T) {
		pub := &fakePublisher{}
		router := NewFrameRouter(pub, topics)
		msg := &datafeedv1.WebsocketWrapMessageChannel{Message: &datafeedv1.WebsocketWrapMessageChannel_Liveprice{Liveprice: &datafeedv1.LivePrice{}}}
		require.NoError(t, router.Route(context.Background(), msg))
		assert.Empty(t, pub.topics)
	})

	t.Run("publisher error propagates", func(t *testing.T) {
		pub := &errorPublisher{}
		router := NewFrameRouter(pub, topics)
		msg := &datafeedv1.WebsocketWrapMessageChannel{Message: &datafeedv1.WebsocketWrapMessageChannel_RunningTradeBatch{RunningTradeBatch: &datafeedv1.RunningTradeBatch{}}}
		require.Error(t, router.Route(context.Background(), msg))
	})
}

type errorPublisher struct{}

func (e *errorPublisher) Publish(_ context.Context, _ string, _ string, _ []byte) error { return errPublish }
func (e *errorPublisher) Close()                                                       {}

var errPublish = assert.AnError
```

> `WebsocketWrapMessageChannel` oneof field names must match the generated proto (`_RunningTradeBatch`, `_Orderbook`, `_Liveprice`); adjust to the actual generated oneof accessor names if the build complains.

- [ ] **Step 3: Run it to verify it fails**

Run: `cd apps/ws && go test ./internal/delivery/ws/... 2>&1 | head -20`
Expected: compile error — `FrameRouter`/`NewFrameRouter` undefined.

- [ ] **Step 4: Write `router.go`**

```go
package ws

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"

	kafkapkg "github.com/nofendian17/sbterm/apps/ws/internal/infrastructure/kafka"
	consumerv1 "github.com/nofendian17/sbterm/libs/proto/securities/transactional/datafeed/consumer/entity/v1"
	datafeedv1 "github.com/nofendian17/sbterm/libs/proto/securities/transactional/datafeed/v1"
)

// FrameRouter maps decoded datafeed frames onto Kafka by channel type. It is
// the single boundary between the websocket transport and the ingestion
// pipeline.
type FrameRouter struct {
	publisher kafkapkg.Publisher
	topics    kafkapkg.Topics
}

// NewFrameRouter builds a router that publishes running trade batches and
// order book snapshots to the configured topics.
func NewFrameRouter(publisher kafkapkg.Publisher, topics kafkapkg.Topics) *FrameRouter {
	return &FrameRouter{publisher: publisher, topics: topics}
}

// Route publishes every ingested channel present in the frame. Non-ingested
// channels are ignored. A topic keyed by symbol keeps one symbol's frames in a
// single Kafka partition, preserving per-symbol ordering.
func (r *FrameRouter) Route(ctx context.Context, m *datafeedv1.WebsocketWrapMessageChannel) error {
	if batch := m.GetRunningTradeBatch(); batch != nil {
		value, err := proto.Marshal(batch)
		if err != nil {
			return fmt.Errorf("ws: marshal running trade batch: %w", err)
		}
		trades := batch.GetBatch()
		symbol := ""
		if len(trades) > 0 {
			symbol = trades[0].GetStock()
		}
		return r.publisher.Publish(ctx, r.topics.RunningTradeBatch, symbol, value)
	}
	if ob := m.GetOrderbook(); ob != nil {
		value, err := proto.Marshal(ob)
		if err != nil {
			return fmt.Errorf("ws: marshal order book: %w", err)
		}
		return r.publisher.Publish(ctx, r.topics.OrderBook, ob.GetStockCode(), value)
	}
	return nil
}
```

- [ ] **Step 5: Run the test to verify it passes**

Run: `cd apps/ws && go test ./internal/delivery/ws/...`
Expected: PASS.

- [ ] **Step 6: Rewrite `ws.go` — drop QuestDB, add the router**

Replace the current `ws.go` body with:

```go
// Package ws runs one Stockbit datafeed websocket client per configured
// subscription for the lifetime of the server, publishing decoded frames to
// Kafka, and stops them on container shutdown.
package ws

import (
	"context"
	"sync"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/nofendian17/sbterm/apps/ws/internal/infrastructure/stockbit"
	datafeedv1 "github.com/nofendian17/sbterm/libs/proto/securities/transactional/datafeed/v1"
	"github.com/nofendian17/sbterm/libs/stockbit"
	"github.com/nofendian17/sbterm/libs/pkg/log"
)

// Subscription couples a dedicated datafeed websocket client with the channel
// that the connection subscribes to on connect.
type Subscription struct {
	Name    string
	Client  *stockbit.WSClient
	Channel *datafeedv1.WebsocketChannel
}

// Service runs one Stockbit datafeed websocket client per subscription,
// routing every decoded frame to the ingestion pipeline through Kafka, and
// stops them all on container shutdown.
type Service struct {
	subs      []*Subscription
	refresher *stockbit.Refresher
	router    *FrameRouter
	logger    log.Logger

	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
}

// New builds a Service around the configured subscriptions.
func New(subs []*Subscription, refresher *stockbit.Refresher, router *FrameRouter, logger log.Logger) *Service {
	return &Service{subs: subs, refresher: refresher, router: router, logger: logger}
}

// BuildChannel maps a channel config onto the corresponding datafeed channel
// by composing the per-service builders. Empty arrays subscribe nothing on
// that channel.
func BuildChannel(ch config.WSChannelConfig) *datafeedv1.WebsocketChannel {
	return stockbit.MergeWSChannels(
		stockbit.WSChannelWatchlist(ch.Watchlist...),
		stockbit.WSChannelOrderBook(ch.OrderBook...),
		stockbit.WSChannelRunningTrade(ch.RunningTrade...),
		stockbit.WSChannelRunningTradeBatch(ch.RunningTradeBatch...),
		stockbit.WSChannelLiveprice(ch.Liveprice...),
		stockbit.WSChannelIepiev(ch.Iepiev...),
		stockbit.WSChannelIntraday(ch.Intraday...),
		stockbit.WSChannelBestBidOffer(ch.BestBidOffer...),
		stockbit.WSChannelLivepriceV3(ch.LivepriceV3...),
		stockbit.WSChannelOrderBookV3(ch.OrderBookV3...),
		stockbit.WSChannelIntradayV3(ch.IntradayV3...),
	)
}

// wsMessageJSON renders a decoded datafeed frame as JSON for debug logging.
func wsMessageJSON(m *datafeedv1.WebsocketWrapMessageChannel) string {
	out, err := protojson.Marshal(m)
	if err != nil {
		return "?"
	}
	return string(out)
}

// Start dials a websocket connection for every configured subscription in the
// background. It is idempotent.
func (s *Service) Start() {
	if s.cancel != nil {
		return
	}
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.done = make(chan struct{})

	userID, err := s.refresher.UserID(context.Background())
	if err != nil {
		s.logger.Warn("ws: resolve stockbit user id failed", "error", err)
	}

	var wg sync.WaitGroup
	for _, sub := range s.subs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.run(sub, userID)
		}()
	}
	go func() {
		wg.Wait()
		close(s.done)
	}()
}

// run drives one subscription's connection until the shared context is
// cancelled.
func (s *Service) run(sub *Subscription, userID int64) {
	request := stockbit.WSSubscription{UserID: userID, Channel: sub.Channel}

	err := sub.Client.Run(s.ctx, request, func(ctx context.Context, m *datafeedv1.WebsocketWrapMessageChannel) error {
		s.logger.Debug("stockbit ws frame", "subscription", sub.Name, "message", wsMessageJSON(m))
		if err := s.router.Route(ctx, m); err != nil {
			s.logger.Warn("ws: publish frame failed", "subscription", sub.Name, "error", err)
		}
		return nil
	})
	if err != nil {
		s.logger.Warn("stockbit ws client stopped", "subscription", sub.Name, "error", err)
	}
}

// Shutdown cancels the run context and waits for every client to stop.
func (s *Service) Shutdown() error {
	if s.cancel != nil {
		s.cancel()
	}
	if s.done != nil {
		select {
		case <-s.done:
		case <-time.After(5 * time.Second):
			s.logger.Warn("stockbit ws clients did not stop within 5s")
		}
	}
	return nil
}
```

> Note the two distinct `stockbit` imports: `apps/ws/internal/infrastructure/stockbit` (WSClient, WSSubscription, MergeWSChannels, WSChannel*) and `libs/stockbit` (Refresher). Alias the local one if the compiler needs help. `BuildChannel` takes `config.WSChannelConfig` from `github.com/nofendian17/sbterm/apps/ws/internal/infrastructure/config` (Task 10), so use that config package in this file.

- [ ] **Step 7: Verify the package builds**

```bash
cd apps/ws && go build ./internal/delivery/ws/...
```

Expected: compiles. (Config import lands in Task 10.)

- [ ] **Step 8: Commit**

```bash
git add apps/ws/internal/delivery/ws
git commit -m "refactor(ws): route datafeed frames to Kafka instead of questdb sinks"
```

### Task 9: ws config package

**Files:**
- Create: `apps/ws/internal/infrastructure/config/config.go`

**Interfaces:**
- Produces:

```go
type Config struct {
	Stockbit StockbitConfig `mapstructure:"stockbit"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Kafka    KafkaConfig    `mapstructure:"kafka"`
	Log      LogConfig      `mapstructure:"log"`
}
type StockbitConfig struct { BaseURL, PlayerID, Username, Password string; WSURL string; WSSubscriptions []WSSubscriptionConfig; WSPingInterval, WSReconnectBackoffInitial, WSReconnectBackoffMax time.Duration; Timeout time.Duration; RetryCount int }
type WSSubscriptionConfig struct { Name string; Channels WSChannelConfig }
type WSChannelConfig struct { Watchlist, OrderBook, RunningTrade, RunningTradeBatch, Liveprice, Iepiev, Intraday, BestBidOffer, LivepriceV3, OrderBookV3, IntradayV3 []string }
type KafkaConfig struct { Brokers []string; RunningTradeBatchTopic string; OrderBookTopic string }
func Load() (*Config, error)
func (c Config) Topics() kafkapkg.Topics
```

- [ ] **Step 1: Write `config.go`** (mirror the api module's structure, trimmed to the ws scope; defaults from `config.yaml.example` ws/stockbit/redis/log sections)

```go
package config

import (
	"errors"
	"time"

	"github.com/spf13/viper"

	kafkapkg "github.com/nofendian17/sbterm/apps/ws/internal/infrastructure/kafka"
)

const (
	ConfigFileName = "config"
	ConfigFileType = "yaml"
	ConfigFilePath = "."
)

// version is overridable at build time via:
// go build -ldflags "-X github.com/nofendian17/sbterm/apps/ws/internal/infrastructure/config.version=<tag>"
var version = "dev"

type Config struct {
	Stockbit StockbitConfig `mapstructure:"stockbit"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Kafka    KafkaConfig    `mapstructure:"kafka"`
	Log      LogConfig      `mapstructure:"log"`
}

type StockbitConfig struct {
	BaseURL                   string                 `mapstructure:"base_url"`
	Timeout                   time.Duration          `mapstructure:"timeout"`
	RetryCount                int                    `mapstructure:"retry_count"`
	PlayerID                  string                 `mapstructure:"player_id"`
	Username                  string                 `mapstructure:"username"`
	Password                  string                 `mapstructure:"password"`
	WSURL                     string                 `mapstructure:"ws_url"`
	WSSubscriptions           []WSSubscriptionConfig `mapstructure:"ws_subscriptions"`
	WSPingInterval            time.Duration          `mapstructure:"ws_ping_interval"`
	WSReconnectBackoffInitial time.Duration          `mapstructure:"ws_reconnect_backoff_initial"`
	WSReconnectBackoffMax     time.Duration          `mapstructure:"ws_reconnect_backoff_max"`
}

type WSSubscriptionConfig struct {
	Name     string          `mapstructure:"name"`
	Channels WSChannelConfig `mapstructure:"channels"`
}

type WSChannelConfig struct {
	Watchlist         []string `mapstructure:"watchlist"`
	OrderBook         []string `mapstructure:"order_book"`
	RunningTrade      []string `mapstructure:"running_trade"`
	RunningTradeBatch []string `mapstructure:"running_trade_batch"`
	Liveprice         []string `mapstructure:"liveprice"`
	Iepiev            []string `mapstructure:"iepiev"`
	Intraday          []string `mapstructure:"intraday"`
	BestBidOffer      []string `mapstructure:"best_bid_offer"`
	LivepriceV3       []string `mapstructure:"liveprice_v3"`
	OrderBookV3       []string `mapstructure:"order_book_v3"`
	IntradayV3        []string `mapstructure:"intraday_v3"`
}

type RedisConfig struct {
	URL string `mapstructure:"url"`
}

type KafkaConfig struct {
	Brokers               []string `mapstructure:"brokers"`
	RunningTradeBatchTopic string  `mapstructure:"running_trade_batch_topic"`
	OrderBookTopic         string  `mapstructure:"order_book_topic"`
}

type LogConfig struct {
	Level     string `mapstructure:"level"`
	Format    string `mapstructure:"format"`
	AddSource bool   `mapstructure:"add_source"`
}

// Topics maps the configured topic names onto the pipeline topic set.
func (c Config) Topics() kafkapkg.Topics {
	return kafkapkg.Topics{RunningTradeBatch: c.Kafka.RunningTradeBatchTopic, OrderBook: c.Kafka.OrderBookTopic}
}

func Load() (*Config, error) {
	v := viper.New()
	setDefaults(v)

	v.SetConfigName(ConfigFileName)
	v.SetConfigType(ConfigFileType)
	v.AddConfigPath(ConfigFilePath)

	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return nil, errors.New("config: read config file: " + err.Error())
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, errors.New("config: unmarshal: " + err.Error())
	}
	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("stockbit.base_url", "https://exodus.stockbit.com")
	v.SetDefault("stockbit.timeout", 30*time.Second)
	v.SetDefault("stockbit.retry_count", 3)
	v.SetDefault("stockbit.ws_url", "wss://wss-trading.stockbit.com/ws")
	v.SetDefault("stockbit.ws_ping_interval", 30*time.Second)
	v.SetDefault("stockbit.ws_reconnect_backoff_initial", time.Second)
	v.SetDefault("stockbit.ws_reconnect_backoff_max", 30*time.Second)
	v.SetDefault("redis.url", "redis://localhost:6379/0")
	v.SetDefault("kafka.brokers", []string{"localhost:29092"})
	v.SetDefault("kafka.running_trade_batch_topic", "datafeed.running_trade_batch")
	v.SetDefault("kafka.order_book_topic", "datafeed.order_book")
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "text")
	v.SetDefault("log.add_source", false)
}
```

- [ ] **Step 2: Verify it compiles and add a tiny config test** (`config_test.go` — defaults resolve; `Topics()` maps them)

```go
package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load()
	assert.NoError(t, err)

	topics := cfg.Topics()
	assert.Equal(t, "datafeed.running_trade_batch", topics.RunningTradeBatch)
	assert.Equal(t, "datafeed.order_book", topics.OrderBook)
	assert.Equal(t, "wss://wss-trading.stockbit.com/ws", cfg.Stockbit.WSURL)
}
```

Run: `cd apps/ws && go test ./internal/infrastructure/config/...`
Expected: PASS (no `config.yaml` present → defaults).

- [ ] **Step 3: Fix the `ws.go` import for `BuildChannel`**

In `apps/ws/internal/delivery/ws/ws.go`, ensure `BuildChannel`'s `config.WSChannelConfig` argument type resolves to `"github.com/nofendian17/sbterm/apps/ws/internal/infrastructure/config"`.

Then: `cd apps/ws && go build ./internal/delivery/ws/...`

- [ ] **Step 4: Commit**

```bash
git add apps/ws/internal/infrastructure/config
git commit -m "feat(ws): add ws-scoped config package with pipeline topics"
```

### Task 10: ws container + entrypoint

**Files:**
- Create: `apps/ws/internal/container/container.go`
- Create: `apps/ws/cmd/ws/main.go`

**Interfaces:**
- Produces: `github.com/nofendian17/sbterm/apps/ws/internal/container.New(cfg, logger) *do.RootScope` and `container.Run() error`; binary via `cmd/ws`.

- [ ] **Step 1: Write `container.go`**

```go
package container

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/samber/do/v2"

	deliveryws "github.com/nofendian17/sbterm/apps/ws/internal/delivery/ws"
	"github.com/nofendian17/sbterm/apps/ws/internal/infrastructure/config"
	kafkapkg "github.com/nofendian17/sbterm/apps/ws/internal/infrastructure/kafka"
	stockbitws "github.com/nofendian17/sbterm/apps/ws/internal/infrastructure/stockbit"
	"github.com/nofendian17/sbterm/libs/pkg/log"
	"github.com/nofendian17/sbterm/libs/stockbit"
)

const shutdownTimeout = 5 * time.Second

func New(cfg *config.Config, logger log.Logger) *do.RootScope {
	injector := do.New()
	do.ProvideValue(injector, cfg)
	do.ProvideValue(injector, logger)

	do.Provide(injector, func(i do.Injector) (*stockbit.Refresher, error) {
		opts := []stockbit.Option{
			stockbit.WithTimeout(cfg.Stockbit.Timeout),
			stockbit.WithRetryCount(cfg.Stockbit.RetryCount),
			stockbit.WithLogger(logger),
		}
		if cfg.Stockbit.BaseURL != "" {
			opts = append(opts, stockbit.WithBaseURL(cfg.Stockbit.BaseURL))
		}
		client := stockbit.New(opts...)

		store := stockbit.NewRedisTokenStore(redisCmd(i))
		refresher := stockbit.NewRefresher(client, store, stockbit.Credentials{
			PlayerID: cfg.Stockbit.PlayerID,
			Username: cfg.Stockbit.Username,
			Password: cfg.Stockbit.Password,
		}, logger)
		client.SetAuthenticator(refresher)
		return refresher, nil
	})

	do.Provide(injector, func(i do.Injector) (*kafkapkg.Producer, error) {
		return kafkapkg.NewProducer(cfg.Kafka.Brokers, logger)
	})

	do.Provide(injector, func(i do.Injector) (*deliveryws.Service, error) {
		refresher, err := do.Invoke[*stockbit.Refresher](i)
		if err != nil {
			return nil, err
		}
		publisher, err := do.Invoke[*kafkapkg.Producer](i)
		if err != nil {
			return nil, err
		}

		subs := make([]*deliveryws.Subscription, 0, len(cfg.Stockbit.WSSubscriptions))
		for _, sub := range cfg.Stockbit.WSSubscriptions {
			ws := stockbitws.NewWSClient(cfg.Stockbit.WSURL, func(ctx context.Context) (string, error) {
				key, err := refresher.Client().GetWebSocketKey(ctx)
				if err != nil {
					return "", fmt.Errorf("ws: fetch websocket key: %w", err)
				}
				return key.Data.Key, nil
			},
				stockbitws.WithWSAccessTokenProvider(func(ctx context.Context) (string, error) {
					return refresher.EnsureToken(ctx)
				}),
				stockbitws.WithWSPingInterval(cfg.Stockbit.WSPingInterval),
				stockbitws.WithWSReconnectBackoff(cfg.Stockbit.WSReconnectBackoffInitial, cfg.Stockbit.WSReconnectBackoffMax),
				stockbitws.WithWSLogger(logger),
			)
			subs = append(subs, &deliveryws.Subscription{
				Name:    sub.Name,
				Client:  ws,
				Channel: deliveryws.BuildChannel(sub.Channels),
			})
		}

		router := deliveryws.NewFrameRouter(publisher, cfg.Topics())
		return deliveryws.New(subs, refresher, router, logger), nil
	})

	return injector
}

// redisCmd builds the redis cmable from the configured URL. The token store
// needs only the connection; failures surface lazily on refresh.
func redisCmd(cfg *config.Config) interface{ Cmdable() ... } { /* see below */ }
```

> `libs/stockbit.NewRedisTokenStore(client redis.Cmdable)` takes the go-redis `Cmdable`. Build a minimal `*redis.Client` directly with `github.com/redis/go-redis/v9` (the api module's cache wrapper is api-internal and cannot be imported):

```go
func redisClient(cfg *config.Config) redis.Cmdable {
	opt, err := redis.ParseURL(cfg.Redis.URL)
	if err != nil {
		return nil
	}
	return redis.NewClient(opt)
}
```

> In `provideStockbit`, pass `redisClient(cfg)` to `stockbit.NewRedisTokenStore(...)`. Add the `github.com/redis/go-redis/v9` dependency to the ws module. Since the token store only holds the `Cmdable`, a plain client with no pool tuning is fine; a nil result surfaces as a refresh error, matching the api's lazy-failure behavior.

- [ ] **Step 2: Write `cmd/ws/main.go`**

```go
package main

import (
	"log"

	"github.com/nofendian17/sbterm/apps/ws/internal/container"
)

func main() {
	if err := container.Run(); err != nil {
		log.Fatal(err)
	}
}
```

- [ ] **Step 3: Write `Run()` + signal handling (append to container.go)**

```go
func Run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	level, lerr := log.ParseLevel(cfg.Log.Level)
	if lerr != nil {
		return lerr
	}
	format, ferr := log.ParseFormat(cfg.Log.Format)
	if ferr != nil {
		return ferr
	}
	logger := log.New(log.WithLevel(level), log.WithFormat(format), log.WithAddSource(cfg.Log.AddSource))
	log.SetDefault(logger)

	injector := New(cfg, logger)

	refresher, err := do.Invoke[*stockbit.Refresher](injector)
	if err != nil {
		return fmt.Errorf("container: construct stockbit refresher: %w", err)
	}
	refresher.Start()

	if len(cfg.Stockbit.WSSubscriptions) == 0 {
		logger.Warn("stockbit ws_subscriptions is empty; no datafeed subscriptions")
	} else {
		wsSvc, err := do.Invoke[*deliveryws.Service](injector)
		if err != nil {
			return fmt.Errorf("container: construct stockbit ws service: %w", err)
		}
		wsSvc.Start()
	}

	awaitSignal(injector, logger)
	return nil
}

func awaitSignal(injector *do.RootScope, logger log.Logger) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, os.Interrupt)
	defer signal.Stop(sigChan)

	sig := <-sigChan
	logger.Info("received shutdown signal", "signal", sig.String())
	if report := injector.Shutdown(); !report.Succeed {
		logger.Error("container shutdown failed", "error", report)
		return
	}
	logger.Info("ws worker stopped")
	_ = errors.New // reserved; remove if unused
}
```

- [ ] **Step 4: Verify it builds**

```bash
cd apps/ws && go mod tidy && go build ./... && go vet ./...
```

- [ ] **Step 5: Commit**

```bash
git add apps/ws/internal/container apps/ws/cmd/ws apps/ws/go.mod apps/ws/go.sum
git commit -m "feat(ws): container wiring, entrypoint, and signal-aware shutdown"
```

---

## Phase 6 — apps/ingest

### Task 11: Move questdb into apps/ingest

**Files:**
- Move: `internal/infrastructure/questdb/` → `apps/ingest/internal/infrastructure/questdb/`
- Modify: its imports (proto, log paths)

**Interfaces:**
- Produces: `github.com/nofendian17/sbterm/apps/ingest/internal/infrastructure/questdb` with `New(ctx, conf, table string, logger, opts...) (*Client, error)`, `Client.NewRunningTradeBatchSink(ctx) (RunningTradeBatchSink, error)`, `Client.NewOrderBookSink(ctx) (OrderBookSink, error)`, `WithOrderBookTable`, `Ping`, `HealthCheck`, `Shutdown`. Sink interfaces `RunningTradeBatchSink{Store(ctx,*datafeedv1.RunningTradeBatch); Close(ctx)}`, `OrderBookSink{Store(ctx,*consumerv1.Orderbook); Close(ctx)}`.

- [ ] **Step 1: Move and rewrite imports**

```bash
git mv internal/infrastructure/questdb apps/ingest/internal/infrastructure/questdb
cd apps/ingest
grep -rl 'sbterm-server/internal/infrastructure/stockbit/proto/' . | xargs -r perl -pi -e 's#github\.com/nofendian17/sbterm-server/internal/infrastructure/stockbit/proto/#github.com/nofendian17/sbterm/libs/proto/#g'
grep -rl 'sbterm-server/pkg/' . | xargs -r perl -pi -e 's#github\.com/nofendian17/sbterm-server/pkg/#github.com/nofendian17/sbterm/libs/pkg/#g'
grep -rl 'sbterm-server/internal/' . | xargs -r perl -pi -e 's#github\.com/nofendian17/sbterm-server/internal/#github.com/nofendian17/sbterm/apps/ingest/internal/#g'
```

- [ ] **Step 2: Tidy and verify**

```bash
cd apps/ingest && go mod tidy && go test ./internal/infrastructure/questdb/... && go vet ./internal/infrastructure/questdb/...
```

Expected: questdb tests green (they exercise the sinks against a live QuestDB only when `QUESTDB_URL` is set; otherwise they pass with the lazy client).

- [ ] **Step 3: Commit**

```bash
git add apps/ingest
git commit -m "refactor(monorepo): move questdb infrastructure into apps/ingest module"
```

### Task 12: Kafka consumer wrapper (ingest)

**Files:**
- Create: `apps/ingest/internal/infrastructure/kafka/consumer.go`

**Interfaces:**
- Produces:

```go
type Consumer struct { client *kgo.Client }
func NewConsumer(brokers []string, group string, topics []string) (*Consumer, error)
func (c *Consumer) PollFetches(ctx context.Context) kgo.Fetches
func (c *Consumer) AllowRebalance()
func (c *Consumer) Close()
```

- [ ] **Step 1: Write `consumer.go`**

```go
package kafka

import (
	"context"
	"fmt"

	"github.com/twmb/franz-go/pkg/kgo"
)

// Consumer reads the datafeed pipeline topics inside one consumer group.
type Consumer struct {
	client *kgo.Client
}

// NewConsumer builds a consumer-group client for the pipeline topics.
func NewConsumer(brokers []string, group string, topics []string) (*Consumer, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ConsumerGroup(group),
		kgo.ConsumeTopics(topics...),
	)
	if err != nil {
		return nil, fmt.Errorf("kafka: new consumer: %w", err)
	}
	return &Consumer{client: client}, nil
}

// PollFetches blocks until a fetch is available or ctx is cancelled.
func (c *Consumer) PollFetches(ctx context.Context) kgo.Fetches {
	return c.client.PollFetches(ctx)
}

// AllowRebalance marks the current batch processed before the group rebalances.
func (c *Consumer) AllowRebalance() {
	c.client.AllowRebalance()
}

// Close leaves the group and closes the client.
func (c *Consumer) Close() {
	c.client.Close()
}
```

- [ ] **Step 2: Verify it compiles**

```bash
cd apps/ingest && go mod tidy && go build ./internal/infrastructure/kafka/...
```

- [ ] **Step 3: Commit**

```bash
git add apps/ingest/internal/infrastructure/kafka apps/ingest/go.mod apps/ingest/go.sum
git commit -m "feat(ingest): add franz-go kafka consumer wrapper"
```

### Task 13: Ingest handler + service (TDD)

**Files:**
- Create: `apps/ingest/internal/service/handler.go`
- Create: `apps/ingest/internal/service/handler_test.go`
- Create: `apps/ingest/internal/service/ingest.go`

**Interfaces:**
- Consumes: `questdb.RunningTradeBatchSink`, `questdb.OrderBookSink`, `kafka.Consumer`.
- Produces:

```go
type Topics struct{ RunningTradeBatch, OrderBook string }
type FrameHandler struct{ runningSink questdb.RunningTradeBatchSink; obSink questdb.OrderBookSink; topics Topics; logger log.Logger }
func NewFrameHandler(runningSink questdb.RunningTradeBatchSink, obSink questdb.OrderBookSink, topics Topics, logger log.Logger) *FrameHandler
func (h *FrameHandler) Handle(ctx context.Context, topic string, value []byte) error
func (h *FrameHandler) Close(ctx context.Context) error

type Service struct{ consumer *kafka.Consumer; handler *FrameHandler; logger log.Logger ... }
func NewService(consumer *kafka.Consumer, handler *FrameHandler, logger log.Logger) *Service
func (s *Service) Start()
func (s *Service) Shutdown() error
```

- [ ] **Step 1: Write the failing handler test** (`handler_test.go`)

```go
package service

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/nofendian17/sbterm/apps/ingest/internal/service"
	consumerv1 "github.com/nofendian17/sbterm/libs/proto/securities/transactional/datafeed/consumer/entity/v1"
	datafeedv1 "github.com/nofendian17/sbterm/libs/proto/securities/transactional/datafeed/v1"
	"github.com/nofendian17/sbterm/libs/pkg/log"
)

type fakeRunningSink struct{ stored []*datafeedv1.RunningTradeBatch; closeErr error }
func (f *fakeRunningSink) Store(_ context.Context, b *datafeedv1.RunningTradeBatch) error { f.stored = append(f.stored, b); return nil }
func (f *fakeRunningSink) Close(context.Context) error { return f.closeErr }

type fakeObSink struct{ stored []*consumerv1.Orderbook; closeErr error }
func (f *fakeObSink) Store(_ context.Context, ob *consumerv1.Orderbook) error { f.stored = append(f.stored, ob); return nil }
func (f *fakeObSink) Close(context.Context) error { return f.closeErr }

func TestFrameHandler(t *testing.T) {
	topics := service.Topics{RunningTradeBatch: "datafeed.running_trade_batch", OrderBook: "datafeed.order_book"}
	logger := log.New(log.WithWriter(io.Discard))

	t.Run("running trade batch topic decodes and stores", func(t *testing.T) {
		rs, os := &fakeRunningSink{}, &fakeObSink{}
		h := service.NewFrameHandler(rs, os, topics, logger)

		bytes, err := proto.Marshal(&datafeedv1.RunningTradeBatch{Batch: []*datafeedv1.RunningTrade{{Stock: "BBRI"}}})
		require.NoError(t, err)

		require.NoError(t, h.Handle(context.Background(), "datafeed.running_trade_batch", bytes))
		require.Len(t, rs.stored, 1)
		assert.Equal(t, "BBRI", rs.stored[0].GetBatch()[0].GetStock())
		require.NoError(t, h.Close(context.Background()))
	})

	t.Run("order book topic decodes and stores", func(t *testing.T) {
		rs, os := &fakeRunningSink{}, &fakeObSink{}
		h := service.NewFrameHandler(rs, os, topics, logger)

		bytes, err := proto.Marshal(&consumerv1.Orderbook{StockCode: "BBCA"})
		require.NoError(t, err)

		require.NoError(t, h.Handle(context.Background(), "datafeed.order_book", bytes))
		require.Len(t, os.stored, 1)
		assert.Equal(t, "BBCA", os.stored[0].GetStockCode())
		require.NoError(t, h.Close(context.Background()))
	})

	t.Run("undecodable record errors", func(t *testing.T) {
		h := service.NewFrameHandler(&fakeRunningSink{}, &fakeObSink{}, topics, logger)
		require.Error(t, h.Handle(context.Background(), "datafeed.running_trade_batch", []byte("not a proto")))
	})

	t.Run("unknown topic errors", func(t *testing.T) {
		h := service.NewFrameHandler(&fakeRunningSink{}, &fakeObSink{}, topics, logger)
		require.Error(t, h.Handle(context.Background(), "datafeed.who", []byte{1}))
	})

	t.Run("sink store error propagates", func(t *testing.T) {
		rs := &errRunningSink{err: errors.New("boom")}
		h := service.NewFrameHandler(rs, &fakeObSink{}, topics, logger)
		bytes, err := proto.Marshal(&datafeedv1.RunningTradeBatch{Batch: []*datafeedv1.RunningTrade{}})
		require.NoError(t, err)
		require.ErrorIs(t, h.Handle(context.Background(), "datafeed.running_trade_batch", bytes), errors.New("boom"))
	})
}

type errRunningSink struct{ fakeRunningSink; err error }
func (e *errRunningSink) Store(_ context.Context, _ *datafeedv1.RunningTradeBatch) error { return e.err }
```

- [ ] **Step 2: Run it to verify it fails**

Run: `cd apps/ingest && go test ./internal/service/... 2>&1 | head -20`
Expected: compile error — `service.FrameHandler` undefined.

- [ ] **Step 3: Write `handler.go`**

```go
package service

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"

	"github.com/nofendian17/sbterm/apps/ingest/internal/infrastructure/questdb"
	consumerv1 "github.com/nofendian17/sbterm/libs/proto/securities/transactional/datafeed/consumer/entity/v1"
	datafeedv1 "github.com/nofendian17/sbterm/libs/proto/securities/transactional/datafeed/v1"
	"github.com/nofendian17/sbterm/libs/pkg/log"
)

// Topics names the Kafka topics the handler understands. Values come from
// config and must match what the ws service publishes.
type Topics struct {
	RunningTradeBatch string
	OrderBook         string
}

// FrameHandler decodes one Kafka record and persists it to the matching
// QuestDB sink. Both sinks are single-writer; Handler is not safe for
// concurrent use.
type FrameHandler struct {
	runningSink questdb.RunningTradeBatchSink
	obSink      questdb.OrderBookSink
	topics      Topics
	logger      log.Logger
}

// NewFrameHandler builds a handler bound to the two sinks.
func NewFrameHandler(runningSink questdb.RunningTradeBatchSink, obSink questdb.OrderBookSink, topics Topics, logger log.Logger) *FrameHandler {
	return &FrameHandler{runningSink: runningSink, obSink: obSink, topics: topics, logger: logger}
}

// Handle routes one record by topic, unmarshalling the protobuf payload and
// storing it.
func (h *FrameHandler) Handle(ctx context.Context, topic string, value []byte) error {
	switch topic {
	case h.topics.RunningTradeBatch:
		batch := &datafeedv1.RunningTradeBatch{}
		if err := proto.Unmarshal(value, batch); err != nil {
			return fmt.Errorf("ingest: decode running trade batch: %w", err)
		}
		return h.runningSink.Store(ctx, batch)
	case h.topics.OrderBook:
		ob := &consumerv1.Orderbook{}
		if err := proto.Unmarshal(value, ob); err != nil {
			return fmt.Errorf("ingest: decode order book: %w", err)
		}
		return h.obSink.Store(ctx, ob)
	default:
		return fmt.Errorf("ingest: unexpected topic %q", topic)
	}
}

// Close flushes and releases both sinks.
func (h *FrameHandler) Close(ctx context.Context) error {
	if err := h.runningSink.Close(ctx); err != nil {
		return fmt.Errorf("ingest: close running trade sink: %w", err)
	}
	if err := h.obSink.Close(ctx); err != nil {
		return fmt.Errorf("ingest: close order book sink: %w", err)
	}
	return nil
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd apps/ingest && go test ./internal/service/...`
Expected: PASS.

- [ ] **Step 5: Write `ingest.go` (the PollFetches loop)**

```go
package service

import (
	"context"
	"sync"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/nofendian17/sbterm/apps/ingest/internal/infrastructure/kafka"
	"github.com/nofendian17/sbterm/libs/pkg/log"
)

// Service drains the pipeline topics and persists each record through the
// FrameHandler until Shutdown. Records that fail to decode or persist are
// logged and redelivered thanks to at-least-once semantics and QuestDB dedup.
type Service struct {
	consumer *kafka.Consumer
	handler  *FrameHandler
	logger   log.Logger

	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
}

// NewService builds the ingestion loop.
func NewService(consumer *kafka.Consumer, handler *FrameHandler, logger log.Logger) *Service {
	return &Service{consumer: consumer, handler: handler, logger: logger}
}

// Start launches the poll loop. It is idempotent.
func (s *Service) Start() {
	if s.cancel != nil {
		return
	}
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.done = make(chan struct{})
	go func() {
		defer close(s.done)
		s.run()
	}()
}

func (s *Service) run() {
	for {
		fetches := s.consumer.PollFetches(s.ctx)
		if fetches.IsClientClosed() {
			return
		}
		if errs := fetches.Errors(); len(errs) > 0 {
			for _, err := range errs {
				s.logger.Warn("kafka: fetch error", "error", err.Err)
			}
		}
		fetches.EachRecord(func(rec *kgo.Record) {
			if err := s.handler.Handle(s.ctx, rec.Topic, rec.Value); err != nil {
				s.logger.Warn("ingest: handle record", "topic", rec.Topic, "partition", rec.Partition, "offset", rec.Offset, "error", err)
			}
		})
		s.consumer.AllowRebalance()
	}
}

// Shutdown stops the poll loop and closes the handler sinks.
func (s *Service) Shutdown() error {
	if s.cancel != nil {
		s.cancel()
	}
	if s.done != nil {
		select {
		case <-s.done:
		case <-time.After(5 * time.Second):
			s.logger.Warn("ingest: poll loop did not stop within 5s")
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.handler.Close(ctx)
}
```

- [ ] **Step 6: Verify the package builds**

```bash
cd apps/ingest && go build ./internal/service/... && go vet ./...
```

- [ ] **Step 7: Commit**

```bash
git add apps/ingest/internal/service
git commit -m "feat(ingest): decode kafka records and persist through questdb sinks"
```

### Task 14: ingest config + container + entrypoint

**Files:**
- Create: `apps/ingest/internal/infrastructure/config/config.go`
- Create: `apps/ingest/internal/container/container.go`
- Create: `apps/ingest/cmd/ingest/main.go`

**Interfaces:**
- Produces: `github.com/nofendian17/sbterm/apps/ingest/internal/infrastructure/config` (`Config`, `Load()`), `.../ingest/internal/container.Run() error`, `container.New(cfg, logger)`, binary via `cmd/ingest`.

- [ ] **Step 1: Write `config.go`**

```go
package config

import (
	"errors"
	"time"

	"github.com/spf13/viper"

	"github.com/nofendian17/sbterm/apps/ingest/internal/service"
)

const (
	ConfigFileName = "config"
	ConfigFileType = "yaml"
	ConfigFilePath = "."
)

// version is overridable at build time via:
// go build -ldflags "-X github.com/nofendian17/sbterm/apps/ingest/internal/infrastructure/config.version=<tag>"
var version = "dev"

type Config struct {
	QuestDB QuestDBConfig `mapstructure:"questdb"`
	Kafka   KafkaConfig   `mapstructure:"kafka"`
	Log     LogConfig     `mapstructure:"log"`
}

type QuestDBConfig struct {
	URL            string `mapstructure:"url"`
	Table          string `mapstructure:"table"`
	OrderBookTable string `mapstructure:"order_book_table"`
}

type KafkaConfig struct {
	Brokers []string `mapstructure:"brokers"`
	Group   string   `mapstructure:"group"`
	Topics  []string `mapstructure:"topics"`
}

type LogConfig struct {
	Level     string `mapstructure:"level"`
	Format    string `mapstructure:"format"`
	AddSource bool   `mapstructure:"add_source"`
}

// Topics returns the service topic set derived from the configured topic list.
func (c Config) Topics() service.Topics {
	topics := service.Topics{}
	for _, t := range c.Kafka.Topics {
		switch t {
		case topicRunningTradeBatch:
			topics.RunningTradeBatch = t
		case topicOrderBook:
			topics.OrderBook = t
		}
	}
	return topics
}

const (
	topicRunningTradeBatch = "datafeed.running_trade_batch"
	topicOrderBook         = "datafeed.order_book"
)

func Load() (*Config, error) {
	v := viper.New()
	setDefaults(v)

	v.SetConfigName(ConfigFileName)
	v.SetConfigType(ConfigFileType)
	v.AddConfigPath(ConfigFilePath)

	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return nil, errors.New("config: read config file: " + err.Error())
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, errors.New("config: unmarshal: " + err.Error())
	}
	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("questdb.url", "ws::addr=localhost:9000;")
	v.SetDefault("questdb.table", "running_trades")
	v.SetDefault("questdb.order_book_table", "order_books")
	v.SetDefault("kafka.brokers", []string{"localhost:29092"})
	v.SetDefault("kafka.group", "sbterm-ingest")
	v.SetDefault("kafka.topics", []string{topicRunningTradeBatch, topicOrderBook})
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "text")
	v.SetDefault("log.add_source", false)
}
```

- [ ] **Step 2: Write `container.go`**

```go
package container

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/samber/do/v2"

	"github.com/nofendian17/sbterm/apps/ingest/internal/infrastructure/config"
	"github.com/nofendian17/sbterm/apps/ingest/internal/infrastructure/kafka"
	"github.com/nofendian17/sbterm/apps/ingest/internal/infrastructure/questdb"
	"github.com/nofendian17/sbterm/apps/ingest/internal/service"
	"github.com/nofendian17/sbterm/libs/pkg/log"
)

func New(cfg *config.Config, logger log.Logger) *do.RootScope {
	injector := do.New()
	do.ProvideValue(injector, cfg)
	do.ProvideValue(injector, logger)

	do.Provide(injector, func(i do.Injector) (*questdb.Client, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return questdb.New(ctx, cfg.QuestDB.URL, cfg.QuestDB.Table, logger, questdb.WithOrderBookTable(cfg.QuestDB.OrderBookTable))
	})

	do.Provide(injector, func(i do.Injector) (*kafka.Consumer, error) {
		return kafka.NewConsumer(cfg.Kafka.Brokers, cfg.Kafka.Group, cfg.Kafka.Topics)
	})

	do.Provide(injector, func(i do.Injector) (*service.Service, error) {
		qdb := do.MustInvoke[*questdb.Client](i)
		consumer := do.MustInvoke[*kafka.Consumer](i)

		runningSink, err := qdb.NewRunningTradeBatchSink(context.Background())
		if err != nil {
			return nil, fmt.Errorf("container: borrow running trade sink: %w", err)
		}
		obSink, err := qdb.NewOrderBookSink(context.Background())
		if err != nil {
			return nil, fmt.Errorf("container: borrow order book sink: %w", err)
		}

		handler := service.NewFrameHandler(runningSink, obSink, cfg.Topics(), logger)
		return service.NewService(consumer, handler, logger), nil
	})

	return injector
}

func Run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	level, lerr := log.ParseLevel(cfg.Log.Level)
	if lerr != nil {
		return lerr
	}
	format, ferr := log.ParseFormat(cfg.Log.Format)
	if ferr != nil {
		return ferr
	}
	logger := log.New(log.WithLevel(level), log.WithFormat(format), log.WithAddSource(cfg.Log.AddSource))
	log.SetDefault(logger)

	injector := New(cfg, logger)

	svc, err := do.Invoke[*service.Service](injector)
	if err != nil {
		return fmt.Errorf("container: construct ingest service: %w", err)
	}
	svc.Start()
	logger.Info("ingest started",
		"group", cfg.Kafka.Group,
		"topics", cfg.Kafka.Topics,
		"questdb", cfg.QuestDB.URL,
	)

	awaitSignal(injector, logger)
	return nil
}

func awaitSignal(injector *do.RootScope, logger log.Logger) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, os.Interrupt)
	defer signal.Stop(sigChan)

	sig := <-sigChan
	logger.Info("received shutdown signal", "signal", sig.String())
	if report := injector.Shutdown(); !report.Succeed {
		logger.Error("container shutdown failed", "error", report)
		return
	}
	logger.Info("ingest stopped")
}
```

- [ ] **Step 3: Write `cmd/ingest/main.go`**

```go
package main

import (
	"log"

	"github.com/nofendian17/sbterm/apps/ingest/internal/container"
)

func main() {
	if err := container.Run(); err != nil {
		log.Fatal(err)
	}
}
```

- [ ] **Step 4: Tidy and verify**

```bash
cd apps/ingest && go mod tidy && go build ./... && go vet ./... && go test ./...
```

- [ ] **Step 5: Commit**

```bash
git add apps/ingest/internal/infrastructure/config apps/ingest/internal/container apps/ingest/cmd/ingest
git commit -m "feat(ingest): config, container wiring, and entrypoint"
```

---

## Phase 7 — Root config, compose, Docker, Makefile, CI

### Task 15: Per-app config files at repo root

**Files:**
- Create: `config.yaml`, `config.ws.yaml`, `config.ingest.yaml` (+ `.example` copies)
- Delete: old `apps/api/config.yaml(.example)` move is already done in Task 5 — set the api example contents here

**Interfaces:**
- Produces: three working config files matching the three modules' config schemas (Task 5/9/14). Values: Redis/Postgres hosts as in current docker-compose network; Stockbit creds placeholders pulled from env.

- [ ] **Step 1: Rewrite `config.yaml` (api)** — take the current `apps/api/config.yaml.example`, drop `questdb` and all `stockbit.ws_*`/`stockbit.ws_subscriptions`, keep db/redis/stockbit REST + rate_limit/http/log.
- [ ] **Step 2: Write `config.ws.yaml`**

```yaml
stockbit:
  base_url: https://exodus.stockbit.com
  timeout: 30s
  retry_count: 3
  player_id: ${STOCKBIT_PLAYER_ID:-""}
  username: ${STOCKBIT_USERNAME:-""}
  password: ${STOCKBIT_PASSWORD:-""}
  ws_url: wss://wss-trading.stockbit.com/ws
  ws_ping_interval: 30s
  ws_reconnect_backoff_initial: 1s
  ws_reconnect_backoff_max: 30s
  ws_subscriptions:
    - name: running_trade_batch_all
      channels:
        running_trade_batch: ["*"]

redis:
  url: redis://localhost:6379/0

kafka:
  brokers: ["localhost:29092"]
  running_trade_batch_topic: datafeed.running_trade_batch
  order_book_topic: datafeed.order_book

log:
  level: info
  format: text
  add_source: false
```

> Keep a second `order_book_selected` subscription mirroring the current example when order book ingestion is wanted.

- [ ] **Step 3: Write `config.ingest.yaml`**

```yaml
questdb:
  url: ws::addr=localhost:9000;
  table: running_trades
  order_book_table: order_books

kafka:
  brokers: ["localhost:29092"]
  group: sbterm-ingest
  topics: ["datafeed.running_trade_batch", "datafeed.order_book"]

log:
  level: info
  format: text
  add_source: false
```

- [ ] **Step 4: Sync `.example` copies** (`config.yaml.example`, `config.ws.yaml.example`, `config.ingest.yaml.example`), replacing placeholders with comments.
- [ ] **Step 5: Commit**

```bash
git add config.yaml config.yaml.example config.ws.yaml config.ws.yaml.example config.ingest.yaml config.ingest.yaml.example
git commit -m "chore(config): split config per service"
```

### Task 16: Dockerfiles

**Files:**
- Delete: root `Dockerfile`
- Create: `apps/api/Dockerfile`, `apps/ws/Dockerfile`, `apps/ingest/Dockerfile`

- [ ] **Step 1: Write the three Dockerfiles** (identical except name/build target). `apps/api/Dockerfile`:

```dockerfile
# syntax=docker/dockerfile:1

FROM golang:1.26.5-alpine AS builder

WORKDIR /src

RUN apk add --no-cache git ca-certificates

COPY go.work go.work.sum* ./
COPY apps ./apps
COPY libs ./libs

COPY . .

ARG APP_VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -X github.com/nofendian17/sbterm/apps/api/internal/infrastructure/config.version=${APP_VERSION}" \
    -o /out/sbterm-api ./apps/api/cmd/server

FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata curl

WORKDIR /app

COPY --from=builder /out/sbterm-api /usr/local/bin/sbterm-api

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD curl -fsS http://127.0.0.1:8080/health >/dev/null || exit 1

ENTRYPOINT ["sbterm-api"]
```

`apps/ws/Dockerfile`: same builder, binary `/out/sbterm-ws`, target `./apps/ws/cmd/ws`, ldflags `.../apps/ws/internal/infrastructure/config.version`, no EXPOSE/HEALTHCHECK.
`apps/ingest/Dockerfile`: same builder, binary `/out/sbterm-ingest`, target `./apps/ingest/cmd/ingest`, ldflags `.../apps/ingest/internal/infrastructure/config.version`, no EXPOSE/HEALTHCHECK.

> `COPY . .` after the module copies is redundant; keep a single `COPY . .` (context is repo root) — the layered `COPY apps ./apps` + `COPY libs ./libs` before it enables cache reuse.

- [ ] **Step 2: Verify the api image builds**

```bash
docker build -f apps/api/Dockerfile -t sbterm-api:test .
```

- [ ] **Step 3: Commit**

```bash
git rm Dockerfile
git add apps/api/Dockerfile apps/ws/Dockerfile apps/ingest/Dockerfile
git commit -m "build(docker): per-app Dockerfiles built from the go.work workspace"
```

### Task 17: docker-compose with Redpanda

**Files:**
- Rewrite: `docker-compose.yml`

- [ ] **Step 1: Write `docker-compose.yml`**

```yaml
services:
  api:
    build:
      context: .
      dockerfile: apps/api/Dockerfile
      args:
        APP_VERSION: ${APP_VERSION:-dev}
    image: sbterm-api:${APP_VERSION:-dev}
    container_name: sbterm-api
    restart: always
    volumes:
      - ./config.yaml:/app/config.yaml:ro
    ports:
      - "${APP_HOST_PORT:-8080}:8080"
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "curl", "-fsS", "http://127.0.0.1:8080/health"]
      interval: 30s
      timeout: 3s
      retries: 3
      start_period: 10s

  ws:
    build:
      context: .
      dockerfile: apps/ws/Dockerfile
      args:
        APP_VERSION: ${APP_VERSION:-dev}
    image: sbterm-ws:${APP_VERSION:-dev}
    container_name: sbterm-ws
    restart: always
    volumes:
      - ./config.ws.yaml:/app/config.yaml:ro
    depends_on:
      redis:
        condition: service_healthy
      redpanda:
        condition: service_healthy

  ingest:
    build:
      context: .
      dockerfile: apps/ingest/Dockerfile
      args:
        APP_VERSION: ${APP_VERSION:-dev}
    image: sbterm-ingest:${APP_VERSION:-dev}
    container_name: sbterm-ingest
    restart: always
    volumes:
      - ./config.ingest.yaml:/app/config.yaml:ro
    depends_on:
      questdb:
        condition: service_healthy
      redpanda:
        condition: service_healthy

  postgres:
    image: postgres:18-alpine
    container_name: sbterm-postgres
    restart: always
    environment:
      POSTGRES_USER: ${POSTGRES_USER:-postgres}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:-postgres}
      POSTGRES_DB: ${POSTGRES_DB:-sbterm}
    ports:
      - "${POSTGRES_HOST_PORT:-5432}:5432"
    volumes:
      - postgres-data:/var/lib/postgresql
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER:-postgres} -d ${POSTGRES_DB:-sbterm}"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 10s

  redis:
    image: redis:8-alpine
    container_name: sbterm-redis
    restart: always
    ports:
      - "${REDIS_HOST_PORT:-6379}:6379"
    volumes:
      - redis-data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 5s

  questdb:
    image: questdb/questdb:10.0.0
    container_name: sbterm-questdb
    restart: always
    environment:
      QDB_HTTP_USER: ${QDB_HTTP_USER:-questdb}
      QDB_HTTP_PASSWORD: ${QDB_HTTP_PASSWORD:-questdb}
    ports:
      - "${QUESTDB_HOST_PORT:-9000}:9000"
    volumes:
      - questdb-data:/var/lib/questdb
    healthcheck:
      test: ["CMD", "curl", "-fsS", "-u", "${QDB_HTTP_USER:-questdb}:${QDB_HTTP_PASSWORD:-questdb}", "http://127.0.0.1:9000/exec?query=select%201"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 10s

  redpanda:
    image: docker.redpanda.com/redpandadata/redpanda:v24.3.12
    container_name: sbterm-redpanda
    restart: always
    command:
      - redpanda start
      - --overprovisioned
      - --smp 1
      - --memory 1G
      - --reserve-memory 0M
      - --node-id 0
      - --kafka-addr PLAIN://0.0.0.0:29092
      - --advertise-kafka-addr PLAIN://redpanda:29092
      - --set redpanda.auto_create_topics_enabled=true
    ports:
      - "${REDPANDA_HOST_PORT:-29092}:29092"
    volumes:
      - redpanda-data:/var/lib/redpanda/data
    healthcheck:
      test: ["CMD", "rpk", "cluster", "health", "--exit-when-healthy"]
      interval: 10s
      timeout: 5s
      retries: 10
      start_period: 10s

volumes:
  postgres-data:
  redis-data:
  questdb-data:
  redpanda-data:
```

> In-container broker host must be `redpanda:29092`; update the compose-mounted config files accordingly (they are authoritative in docker).

- [ ] **Step 2: Validate**

```bash
docker compose config --quiet
```

Expected: no schema errors.

- [ ] **Step 3: Commit**

```bash
git add docker-compose.yml
git commit -m "feat(compose): add redpanda and split app into api/ws/ingest services"
```

### Task 18: Makefile, CI, pre-commit, README

**Files:**
- Rewrite: `Makefile`
- Modify: `.github/workflows/test.yml`
- (pre-commit hook unchanged — verifies)
- Modify: `README.md`

- [ ] **Step 1: Rewrite `Makefile`**

```makefile
.PHONY: help run-api run-ws run-ingest build test test-race vet fmt fmt-check install-hooks mock tidy

GO_FILES := $(shell find . -name '*.go' -not -path './.git/*')

help: ## List available targets
	@grep -E '^[a-zA-Z_-]+:.*?## ' Makefile | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "%-14s %s\n", $$1, $$2}'

run-api: ## Run the REST api
	go run ./apps/api/cmd/server

run-ws: ## Run the datafeed websocket publisher
	go run ./apps/ws/cmd/ws

run-ingest: ## Run the questdb ingester
	go run ./apps/ingest/cmd/ingest

build: ## Build every binary into bin/
	mkdir -p bin
	go build -o bin/sbterm-api ./apps/api/cmd/server
	go build -o bin/sbterm-ws ./apps/ws/cmd/ws
	go build -o bin/sbterm-ingest ./apps/ingest/cmd/ingest

test: ## Run all tests in the workspace
	go test ./...

test-race: ## Run all tests with the race detector
	go test -race ./...

vet: ## Run go vet on the workspace
	go vet ./...

fmt: ## Format all Go source files with gofmt
	gofmt -w $(GO_FILES)

fmt-check: ## Fail if any Go file is not gofmt-formatted
	@unformatted=$$(gofmt -l $(GO_FILES)); \
	if [ -n "$$unformatted" ]; then \
		echo "gofmt: the following files need formatting (run 'make fmt'):"; \
		echo "$$unformatted"; \
		exit 1; \
	fi; \
	echo "gofmt: all files formatted"

install-hooks: ## Install git hooks (core.hooksPath -> .githooks)
	chmod +x .githooks/pre-commit
	git config core.hooksPath .githooks
	@echo "git hooks installed (core.hooksPath = $$(git config core.hooksPath))"

mock: ## Generate mocks with uber-go/mock (go generate)
	go generate ./apps/api/...

tidy: ## Tidy every module's go.mod and go.sum
	go work sync
	(cd apps/api && go mod tidy)
	(cd apps/ws && go mod tidy)
	(cd apps/ingest && go mod tidy)
	(cd libs/pkg && go mod tidy)
	(cd libs/proto && go mod tidy)
	(cd libs/stockbit && go mod tidy)
```

- [ ] **Step 2: Update `.github/workflows/test.yml`** — replace the `Install dependencies` step and the test command:

```yaml
      - name: Install dependencies
        run: go work sync

      - name: Check gofmt formatting
        run: make fmt-check

      - name: Run tests with race detector
        run: go test -race -v ./...
```

- [ ] **Step 3: Verify workspace-wide checks**

```bash
make fmt-check && make vet && make test
```

Expected: green — this is the first full-workspace run.

- [ ] **Step 4: Update `README.md`** — replace the structure block and commands with the new monorepo layout, three config files, and the pipeline description (ws → Redpanda → ingest → QuestDB). Mention Redpanda + franz-go in the tech stack.

- [ ] **Step 5: Commit**

```bash
git add Makefile .github/workflows/test.yml README.md
git commit -m "chore: workspace-wide make targets, CI, and monorepo README"
```

### Task 19: Final verification (end-to-end smoke)

**Files:** none (verification only)

- [ ] **Step 1: Full workspace check**

```bash
make fmt-check && make vet && make test-race
```

Expected: all modules' tests pass with the race detector.

- [ ] **Step 2: Compile every binary**

```bash
make build
```

Expected: `bin/sbterm-api`, `bin/sbterm-ws`, `bin/sbterm-ingest`.

- [ ] **Step 3: Compose up**

```bash
docker compose up -d --build
```

Expected: api/ws/ingest/postgres/redis/questdb/redpanda all healthy.

- [ ] **Step 4: Verify ingestion path end-to-end**

With real Stockbit creds in `config.ws.yaml`:

```bash
docker compose logs -f ws ingest
# then, on the questdb http console (localhost:9000):
#   SELECT count() FROM running_trades;
```

Expected: the ws container logs decoded frames with no producer errors; ingest logs no handler errors; `running_trades`/`order_books` row counts increase.

- [ ] **Step 5: Commit any fixes surfaced by the smoke test**

```bash
git add -A
git commit -m "chore: fixes surfaced by end-to-end smoke test"
```