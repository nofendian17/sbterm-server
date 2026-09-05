# `stocks` Feature in `apps/core` — Design

**Date:** 2026-09-05
**Status:** Approved (post-brainstorming)
**Path:** Bounded — vertical slice inside `apps/core`, follows the existing module pattern.

## Goal

Give `apps/core` an admin-managed stock catalog. Authenticated users fetch; admins manage. Out of scope: background sync, real-time prices, cross-service changes.

## Boundaries

- `apps/core` only. No changes to `apps/api`, `apps/ws`, `apps/ingest`, `apps/stream`.
- `apps/core` calls `apps/api` (an internal service-to-service call) for sync. `apps/api` is unchanged.
- The stock catalog is a **new** concept in `apps/core` — distinct from `apps/api`'s read-through stocks endpoint, which still exists and stays untouched.

## Data model

New table `stocks` (migration `000004_create_stocks`):

| column       | type           | notes                                                |
|--------------|----------------|------------------------------------------------------|
| `symbol`     | TEXT PK        | Stockbit ticker (`BBCA`, `TLKM`, …)                  |
| `name`       | TEXT NOT NULL  | Display name from Stockbit                            |
| `sector`     | TEXT           | nullable                                              |
| `exchange`   | TEXT           | nullable                                              |
| `is_active`  | BOOLEAN NOT NULL DEFAULT true | admin toggle                            |
| `synced_at`  | TIMESTAMPTZ    | last successful sync (nullable on insert)             |
| `created_at` | TIMESTAMPTZ NOT NULL DEFAULT now() |                                              |
| `updated_at` | TIMESTAMPTZ NOT NULL DEFAULT now() | trigger from migration 000001             |
| `deleted_at` | TIMESTAMPTZ    | soft delete (consistent with users/watchlists)        |

Triggers reuse the `set_updated_at()` function from migration 000001.

Indexes:
- `idx_stocks_active ON stocks(is_active) WHERE deleted_at IS NULL`
- `idx_stocks_sector ON stocks(sector) WHERE deleted_at IS NULL`

**No `metadata` JSONB column.** `apps/api` is a stateless proxy; storing the upstream payload would be wasted bytes.

## RBAC

Three new permissions, seeded idempotently and granted to roles in the same migration:

| name           | granted to    | used by                                              |
|----------------|---------------|------------------------------------------------------|
| `stocks:read`  | `user`, `admin` | GET `/api/v1/stocks`, GET `/api/v1/stocks/{symbol}` |
| `stocks:write` | `admin` only   | POST/PATCH/DELETE `/api/v1/admin/stocks[/{symbol}]` |
| `stocks:sync`  | `admin` only   | POST `/api/v1/admin/stocks/sync`                     |

## Module layout (apps/core only)

```
apps/core/internal/
  domain/stock.go                        # Stock struct, filter/patch types, StockSyncResult
  repository/stock.go                    # StockRepository port
  repository/stock_sync_client.go        # StockSyncClient port
  infrastructure/repository/stock.go     # pgx implementation
  infrastructure/stockapi/client.go      # HTTP adapter that calls apps/api
  usecase/stock.go                       # StockUsecase
  delivery/http/stock/handler.go         # user-facing read endpoints
  delivery/http/admin/handler.go         # + 4 admin methods (Create/Update/Delete/Sync)
  delivery/http/router.go                # + 6 routes
  infrastructure/config/config.go        # + StockbitAPIConfig{BaseURL,Timeout}
  container/container.go                 # + providers for the new deps
  mocks/                                 # regenerated via go:generate
migrations/core/000004_create_stocks.{up,down}.sql
```

## Repository port

```go
type StockRepository interface {
    GetBySymbol(ctx context.Context, symbol string) (domain.Stock, error)        // ErrStockNotFound
    List(ctx context.Context, filter domain.StockFilter) ([]domain.Stock, int, error)
    Upsert(ctx context.Context, s domain.Stock) (created bool, err error)
    Create(ctx context.Context, s domain.Stock) error                            // ErrStockSymbolTaken
    Update(ctx context.Context, symbol string, patch domain.StockPatch) error
    SoftDelete(ctx context.Context, symbol string) error
}

type StockFilter struct {
    Query    string
    Sector   string
    IsActive *bool   // pointer so "unset" differs from "false"
    Page     int
    Limit    int
}
```

Every read filters `deleted_at IS NULL`. The repository handles pagination defaults (page=1, limit=20, cap=100) to match the existing admin user list.

`Upsert` returns `(xmax = 0) AS created` from the SQL — the standard pgx trick to distinguish INSERT from UPDATE. The `DO UPDATE … WHERE` clause ensures unchanged rows are skipped (pgx returns `sql.ErrNoRows`, which the repository maps to `(false, nil)`).

## StockSyncClient port

```go
type StockSyncClient interface {
    ListSymbols(ctx context.Context) ([]domain.Stock, error)
}
```

The concrete implementation is an HTTP client in `infrastructure/stockapi` that calls `GET {baseURL}/api/v1/stocks` on `apps/api`. Each response row maps to `domain.Stock{IsActive: true, SyncedAt: now()}`. The client is responsible for the request timeout and must surface upstream errors.

## Usecase

```go
type StockUsecase interface {
    List(ctx context.Context, filter domain.StockFilter) ([]domain.Stock, int, error)
    GetBySymbol(ctx context.Context, symbol string) (domain.Stock, error)
    Create(ctx context.Context, s domain.Stock) (domain.Stock, error)
    Update(ctx context.Context, symbol string, patch domain.StockPatch) error
    SoftDelete(ctx context.Context, symbol string) error
    SyncAll(ctx context.Context) (domain.StockSyncResult, error)
}
```

`SyncAll` is best-effort per symbol: an error on one stock is recorded in `result.Errors` and counted in `result.Failed`, never returned. The function only returns an error when the upstream call itself failed (zero symbols fetched).

`Create` rejects empty symbol/name as `ErrInvalidInput`. `Update` with an all-nil patch is a no-op.

## HTTP routes (added to `internal/delivery/http/router.go`)

```
# user-facing (authenticated, requires stocks:read)
GET    /api/v1/stocks                       ?q=&sector=&active=&page=&limit=
GET    /api/v1/stocks/{symbol}

# admin (within /admin, gated by RequirePermission)
POST   /api/v1/admin/stocks                 stocks:write
PATCH  /api/v1/admin/stocks/{symbol}        stocks:write
DELETE /api/v1/admin/stocks/{symbol}        stocks:write
POST   /api/v1/admin/stocks/sync            stocks:sync
```

`stocks:read` and `stocks:write/sync` are mutually exclusive admin groups. The existing `AuthMiddleware` runs first for all routes (the new user-facing GET endpoints sit inside the existing authenticated group, not at the top level).

## Error handling

Sentinels in `domain/errors.go`:

```go
ErrStockNotFound    = errors.New("stock not found")
ErrStockSymbolTaken = errors.New("stock symbol already exists")
ErrStockSyncFailed  = errors.New("stock sync failed")
```

Mapping (mirrors the existing module):

| sentinel             | HTTP status                |
|----------------------|----------------------------|
| `ErrInvalidInput`    | 422 (validator)            |
| `ErrStockNotFound`   | 404                        |
| `ErrStockSymbolTaken`| 409                        |
| any other            | 500                        |
| upstream sync failure| 502 (`response.CodeUpstreamError`) |

## Configuration

```yaml
stockbit_api:
  base_url: "http://localhost:8080"   # apps/api
  timeout: 30s
```

Defaults live in `config.setDefaults`. `apps/core` is now configured to point at `apps/api`'s address.

## Testing

Unit tests only (per user direction). Patterns:

- **Usecase** — `MockStockRepository` + `MockStockSyncClient` + a tiny `mocksLogger` type that satisfies `log.Logger`. Cover List, GetBySymbol, Create (duplicate path), SyncAll (upstream error path, mixed per-symbol result).
- **Handler (user-facing)** — `MockStockUsecase`. Cover 200/404 + JSON shape.
- **Handler (admin)** — extend `MockAdminUsecase` tests with `MockStockUsecase` for the 4 new methods.
- **Repository (pgx)** — `pgxmock`. Cover `GetBySymbol` not-found/found, `Create` unique-violation mapping. Reuse the existing `isUniqueViolation` helper.
- **StockSyncClient (HTTP)** — `httptest.NewServer`. Cover 200/500/invalid JSON.

No live Postgres, Redis, or `apps/api` calls.

## Out of scope

- Background sync loop / scheduler.
- Bulk import endpoint.
- Price/quote caching (lives in `apps/api`).
- Removing or modifying `apps/api`'s `/stocks` endpoint.
- Changes to `apps/ws`, `apps/ingest`, `apps/stream`.
- New microservice.

## Known trade-offs

- **Manual PATCH can be immediately overwritten by the next `/sync` call.** Admins should know sync is authoritative. If this becomes a real problem, add a `manually_edited_at` flag and skip in sync — not in this slice.
- **`apps/core` now depends on `apps/api` being reachable for the sync endpoint.** User endpoints (GET) don't depend on it at all — they only need Postgres, already pinged at startup.
- **`metadata` is dropped.** If we later need to surface fields apps/api returns that we don't persist, we'll add a column then.
- **`StockSyncResult.Skipped` is not populated by `SyncAll` today.** The repository already filters "no change" via the `ON CONFLICT … WHERE` clause. If a caller needs to distinguish "updated" from "no-op", we can add a second return value from `Upsert` later.

## Decisions log

- **2026-09-05** — Drop `metadata JSONB` column. apps/api is a stateless proxy, payload storage would be wasted.
- **2026-09-05** — Three permissions (`read` / `write` / `sync`) instead of one combined. Matches the existing pattern of granular RBAC.
- **2026-09-05** — Sync is admin-triggered only, synchronous, request-scoped. No background loop.
- **2026-09-05** — `apps/core` calls `apps/api` for sync. No new Stockbit auth in `apps/core`.
