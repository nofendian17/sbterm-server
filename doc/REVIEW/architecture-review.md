# sbterm-server — Architecture Review & Refactoring Report

**Date:** 2026-08-19 · **Scope:** Go 1.26.5 workspace monorepo (6 modules) · **Status:** Review complete; fixes applied and verified (uncommitted)

---

## 1. Findings (severity-ranked)

### F-01 — 31-argument router constructor
- **Severity:** High
- **Location:** `apps/api/internal/delivery/http/router.go` — `func NewRouter(h1, h2, ..., h31 *xxx.Handler, logger log.Logger, opts ...RouterOption) chi.Router`; call site `apps/api/internal/container/container.go` (~31 sequential `do.MustInvoke` lines)
- **Problem:** Ten-pable function signature — 31 positional params, error-prone, unreadable, unwieldy to extend.
- **Why it matters:** Every new handler = signature change + call-site churn; the noise buries what the router actually does.
- **Recommendation:** Group handlers into a single `Handlers` struct, one field per domain, and take it as one parameter.
- **Example (applied):**
  ```go
  type Handlers struct {
      Health   *health.HealthHandler
      Trending *trending.TrendingHandler
      // ...one field per handler
  }

  func NewRouter(hs Handlers, logger log.Logger, opts ...RouterOption) chi.Router
  ```
- **Status: ✅ APPLIED** — `router.go`, `container.go`, `router_test.go` (both call sites) rewritten to the struct form. Diff: `router.go` 109±, `container.go` 67±.

### F-02 — Constructor returning an interface
- **Severity:** Medium
- **Location:** `libs/pkg/httpclient/client.go:62` — `func NewClient(opts ...Option) Client`
- **Problem:** Returns the `Client` interface. Only one implementation exists; consumers get an interface with no second implementation, and the interface lives next to the implementation rather than where it's consumed. (Go convention: *accept* interfaces, *return* concrete types.)
- **Why it matters:** `Client`/`Doer` are genuinely useful at *consumption* time (`libs/stockbit` needs only `Do`; tests need to inject a fake). Returning the concrete type keeps that flexibility while making the constructor honest. Users can still assign the result to an interface if they want one.
- **Recommendation:** Return `*client` from `NewClient`; keep the `Client`/`Doer` interfaces exported for consumers and tests. (Note: `client_test.go` and `libs/stockbit` use the interfaces — they still compile unchanged.)
- **Example (applied):**
  ```go
  func NewClient(opts ...Option) *client { ... }
  ```
- **Status: ✅ APPLIED** — verified `go build ./...` across the workspace, tests pass.

### F-03 — `tools.go` blank-import for Go <1.24 (legacy pattern)
- **Severity:** Low
- **Location:** `apps/api/tools.go` (blank-import pins mockgen)
- **Problem:** Go 1.24+ has native `tool` directives in go.mod — the legacy `tools.go` blank-import workaround is obsolete for this Go 1.26 toolchain.
- **Why it matters:** `tool` directives are reproducible (`go install tool`), versioned in go.mod, and visible in the dependency graph. The blank-import file depends on a synthetic `tools` build tag that nothing else uses.
- **Recommendation:** `go get -tool go.uber.org/mock/mockgen`, drop `tools.go`, use `go tool mockgen` in the `//go:generate` directives.
- **Status: ⏸ DEFERRED** (P3) — pure mechanical churn with no behavior change; the repo's `mock` make target (`cd apps/api && go generate ./...`) works today. Do this when the go:generate comments are next touched.

### F-04 — Doc comment attached to the wrong symbol
- **Severity:** Low
- **Location:** `apps/ws/internal/infrastructure/stockbit/wsclient.go` (~lines 504–508), doc comment about `MergeWSChannels` sitting above `sleepCtx`.
- **Problem:** `go doc` for `sleepCtx` showed `MergeWSChannels`'s comment; the actual function had none.
- **Why it matters:** Broken doc attribution misleads `go doc`/IDE hover; a future refactor could move the comment with the wrong function.
- **Recommendation:** Move the comment to `MergeWSChannels`; give `sleepCtx` its own one-liner.
- **Status: ✅ APPLIED** — line 504: `// sleepCtx waits for d or until ctx is done, reporting which happened.`; the merge comment now sits above `MergeWSChannels` (line 528).

### F-05 — Table test with hardcoded BBCA special-case
- **Severity:** Low
- **Location:** `apps/api/internal/delivery/http/stocks/handler_test.go`
- **Problem:** The first table row asserted `CompanyStatus`/`IsUMA` with raw literals inside the loop body while other rows couldn't assert them at all — an implicit "only this row" special case.
- **Why it matters:** Adding a row that should assert these fields required restructuring; the special-case hid the contract (which fields the JSON envelope actually carries).
- **Recommendation:** Add `wantCompanyStatus *string` / `wantIsUMA *bool` table fields; assert only when non-nil (a nil assert-field is the idiomatic "don't check this row" signal).
- **Status: ✅ APPLIED** — both rows now use `ptr(...)` fields; `ptr[T]` helper added at the bottom of the file.

### F-06 — `find` with `-not -path` in Makefile with nothing actually under `./.git`
- **Severity:** Low
- **Location:** `Makefile` — `GO_FILES := $(shell find apps libs -name '*.go' -not -path './.git/*')`
- **Problem:** The `-not -path './.git/*'` clause is dead weight — `find apps libs` never searches `.git` (it starts in `apps` and `libs`).
- **Why it matters:** Dead predicates mislead readers into thinking the tree layout needs guarding; the shorter form is also measurably faster.
- **Recommendation:** Drop the clause.
- **Status: ✅ APPLIED** — `GO_FILES := $(shell find apps libs -name '*.go')`.

### F-07 — CI runs tests but no static analysis
- **Severity:** Low
- **Location:** `.github/workflows/test.yml`
- **Problem:** CI checks gofmt and runs `-race` tests but never runs `go vet`.
- **Why it matters:** vet catches real classes of bugs (unreachable code, bad printf formats, copying locks) that tests often miss, at almost no CI cost.
- **Recommendation:** Add a `make vet` step.
- **Status: ✅ APPLIED** — step `Run go vet` added after `test-race`.

> *Note:* `apps/ws/internal/infrastructure/kafka/producer.go` shows an uncommitted `+kgo.AllowAutoTopicCreation(),` line in the diff. **That edit is not mine** — it appeared in the working tree during this session (concurrent edit by the user or another session). It compiles, passes vet/tests, and I left it untouched.

---

## 2. Architecture assessment

The warehouse layout is sound: three apps (`api`, `ws`, `ingest`) + three libs (`pkg`, `proto`, `stockbit`) in a `go.work` workspace, each app following Clean Architecture (`cmd → container → delivery → usecase → repository → infrastructure`, domain at the center). Good call on:

- consuming Redpanda/Kafka through a `libs/pkg` wrapper — the `franz-go` dependency is quarantined
- versioned protobuf paths (`libs/proto/*/v1`) — breaking changes are opt-in per version
- `samber/do` for the container — the ~103 `do.Provide` graph is explicit, build-time-verifiable wiring

Worth noting, deliberately left as-is:

- **Config duplication across apps** (each reads env directly): explicit over clever; a shared config lib only pays off at 4+ apps. `ponytail:` marker in code.
- **One design gap:** `apps/ws` fetches `UserID` once at startup; if Stockbit rotates the user context (the `KeyProvider` comments already acknowledge rotation), the subscribe frame becomes stale. Extending `KeyProvider` to also supply the user id is the P3 upgrade path. `ponytail:` marker in code.

## 3. Project structure assessment

Strong overall: `cmd/` is thin (parse flags, wire, call Run), `internal/` contains all business logic, nothing in `internal/` is accidentally exported, libraries live under `libs/` (role-named: `pkg`, `proto`, `stockbit`). The one deviation from convention — `delivery/http/{domain}` handler packages instead of a flatter `delivery/http` package — is a *deliberate, good* choice at this scale: 31 handlers × 4 files each would form one unmaintainable package. Keep it.

## 4. Dependency boundary assessment

No circular dependencies — the workspace compiles cleanly, `go mod graph` shows a sane direction (apps → libs, never libs → apps). Two patterns to keep an eye on:

- `libs/stockbit` imports `libs/pkg/httpclient` (a lib → lib edge): fine — `pkg` is the "generic plumbing" layer.
- `libs/stockbit` is shared by `api` *and* `ws`: watch for it accreting app-specific code. If it ever imports an app, that's the moment to split it.

## 5. Shared infrastructure assessment

`libs/pkg/httpclient` is a good example of the quarantine pattern: the heimdall dependency is isolated behind a small surface (`Client`/`Doer`), testable via `WithHTTPClient` injection. Post-fix it also stops returning an interface from a constructor. The remaining shared seams (log, kafka, questdb) follow the same pattern and are consistent with each other.

## 6. Go idiomatic assessment

Couple of stumbling blocks found (F-02 interface-returning constructor, F-03 legacy tools.go) — both fixed or scheduled. Otherwise the codebase is notably idiomatic: MixedCaps everywhere, no stuttering constructors, functional options consistently `With*`, error strings lowercase, `assert.New(t)` per subtest, table-driven tests with named subtests, context-first parameter ordering. No `any` abuse, no premature generics.

## 7. Testing assessment

Good baseline: table-driven handlers tests with named subtests, gomock for usecase mocks, race detector in CI, gofmt check in CI. Improvements applied: the BBCA table special-case became proper assert-fields (F-05). Remaining observations (deliberately **not** applied):

- **Parallelism:** `t.Parallel()` is absent across the suite. Adding it is optional — the suite is already fast; do it when test time becomes a complaint, not before. (P3)
- **Integration tests:** none exist for the Kafka/QuestDB pipeline. That's correct for now — they need a compose stack; the repo already has one under `docker-compose.yml` (per `doc/deployment.md`). Stand up integration tests in the same PR that uses the stack. (P3)

---

## 8. Refactoring plan (P0 → P3)

| # | Change | Prio | Status |
|---|--------|------|--------|
| 1 | Group 31 handler params into a `Handlers` struct (`router.go`, `container.go`, `router_test.go`) | P0 | ✅ APPLIED |
| 2 | `NewClient` returns `*client` (concrete type), keep `Client`/`Doer` interfaces for consumers | P1 | ✅ APPLIED |
| 3 | Fix misplaced doc comment (`wsclient.go`); give `sleepCtx` its own | P1 | ✅ APPLIED |
| 4 | Table-ize the BBCA assert special-case (`stocks/handler_test.go`) | P1 | ✅ APPLIED |
| 5 | Drop dead `-not -path` clause from Makefile `GO_FILES` | P2 | ✅ APPLIED |
| 6 | Add `make vet` to CI | P2 | ✅ APPLIED |
| 7 | Migrate `tools.go` → go.mod `tool` directive for mockgen | P3 | ⏸ DEFERRED — mechanical churn, no behavior change; do when go:generate is next touched |
| 8 | Support user-id rotation in `apps/ws` datafeed client (extend `KeyProvider` → also supply UserID) | P3 | ⏸ DEFERRED — design note; needs Stockbit-side confirmation the id can rotate |
| 9 | `t.Parallel()` across test suites | P3 | ⏸ DEFERRED — suite is fast; add when test time matters |
| 10 | Integration tests for Kafka/QuestDB pipeline (build tag `integration`) | P3 | ⏸ DEFERRED — do together with docker-compose stack usage |

---

## 9. Verification

```
make fmt          ✅ (gofmt -w over all files; no errors)
make fmt-check    ✅ "gofmt: all files formatted"
make build        ✅ bin/sbterm-api, bin/sbterm-ws, bin/sbterm-ingest
make test-race    ✅ all 6 modules pass with -race
make vet          ✅ all 6 modules clean
```

**Changes are NOT committed** — 8 files modified by this review (+199/−89), plus the pre-existing concurrent edit to `producer.go`. Review and commit at your leisure, or ask me to phrase the commit(s).