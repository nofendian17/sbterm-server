# Stocks Feature in `apps/core` — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an admin-managed stock catalog to `apps/core`. Authenticated users can list/get/search; admins can create, update, soft-delete, and trigger a sync from `apps/api`.

**Architecture:** Pure vertical slice inside `apps/core` — `domain.Stock` → `repository.StockRepository` port → pgx impl in `infrastructure/repository` → `usecase.StockUsecase` → HTTP handlers in `delivery/http/stock` (read) and additions to `delivery/http/admin/handler.go` (write/sync). A narrow `StockSyncClient` port lets the usecase call `apps/api` over HTTP without importing any HTTP library; the concrete client lives in `infrastructure/stockapi`. New permissions (`stocks:read` / `stocks:write` / `stocks:sync`) gate routes. Migrations are embedded SQL via the existing `golang-migrate` + `embed.FS` pattern.

**Tech Stack:** Go 1.26, chi/v5, pgx/v5, samber/do/v2, samber/slog-chi, golang-migrate, mockgen, pgxmock, testify, viper.

**Spec:** [docs/superpowers/specs/2026-09-05-stocks-feature-design.md](../specs/2026-09-05-stocks-feature-design.md) (carried alongside this plan; executor reads both).

## Global Constraints

- Go module is `github.com/nofendian17/sbterm/apps/core` (do not introduce new top-level modules).
- `apps/core/internal/repository/Querier` is the only DB seam — every new repository method must take a `Querier`, never `*pgxpool.Pool` or `pgx.Tx` directly.
- All read paths filter `deleted_at IS NULL` (soft delete, consistent with `users` / `watchlists`).
- New permissions are seeded in the same migration via `INSERT ... ON CONFLICT (name) DO NOTHING` and granted to roles the same way migration `000002_create_rbac` does.
- `libs/pkg` is the only cross-module import for utility packages (`log`, `response`, `validator`).
- No changes to `apps/api`, `apps/ws`, `apps/ingest`, `apps/stream`.
- `apps/core`'s port `:8082` and base URL `/api/v1` stay.
- All new tests are unit tests (per user direction); use `pgxmock` for the pgx repository, `httptest.NewServer` for the stockapi client, `testify` + `mockgen` for the rest. No live Postgres / Redis / Stockbit calls in tests.
- Bcrypt cost default is 12; do not change it.

## File Structure

Files created or modified by this plan. Each task says which it touches.

| File | Responsibility |
|---|---|
| `apps/core/internal/domain/stock.go` | `Stock` struct, `StockFilter`/`StockPatch`, sentinel errors. |
| `apps/core/internal/domain/errors.go` | (modify) add `ErrStockNotFound`, `ErrStockSymbolTaken`, `ErrStockSyncFailed`. |
| `apps/core/internal/repository/stock.go` | `StockRepository` port interface + filter/patch types. |
| `apps/core/internal/infrastructure/repository/stock.go` | pgx implementation of `StockRepository`. |
| `apps/core/internal/infrastructure/repository/stock_test.go` | pgxmock unit tests for the pgx impl. |
| `apps/core/internal/usecase/stock.go` | `StockUsecase` interface + private impl. |
| `apps/core/internal/usecase/stock_test.go` | usecase unit tests with mock repo + mock sync client. |
| `apps/core/internal/infrastructure/stockapi/client.go` | `StockSyncClient` HTTP impl that calls `apps/api` `GET /api/v1/stocks`. |
| `apps/core/internal/infrastructure/stockapi/client_test.go` | `httptest.NewServer` tests for the HTTP client. |
| `apps/core/internal/delivery/http/stock/handler.go` | `StockHandler` for user-facing read endpoints. |
| `apps/core/internal/delivery/http/stock/handler_test.go` | unit tests for `StockHandler`. |
| `apps/core/internal/delivery/http/admin/handler.go` | (modify) add `CreateStock`, `UpdateStock`, `DeleteStock`, `SyncStocks` methods. |
| `apps/core/internal/delivery/http/admin/handler_test.go` | (modify) add tests for the four new admin methods. |
| `apps/core/internal/delivery/http/router.go` | (modify) add `Stock` field to `Handlers`, register 6 routes, accept `*stock.StockHandler` in DI. |
| `apps/core/internal/container/container.go` | (modify) provide `repository.StockRepository` (pgx), `repository.StockSyncClient` (stockapi), `usecase.StockUsecase`, `*stock.StockHandler`; pass `*stock.StockHandler` into router. |
| `apps/core/internal/infrastructure/config/config.go` | (modify) add `StockbitAPIConfig{BaseURL, Timeout}` and defaults. |
| `apps/core/migrations/core/000004_create_stocks.up.sql` | new migration: `stocks` table, indexes, triggers, seed perms + role grants. |
| `apps/core/migrations/core/000004_create_stocks.down.sql` | drop table, revert role grants, remove perm rows. |
| `apps/core/internal/mocks/mock_stock_repository.go` | (generated) mockgen output for `StockRepository`. |
| `apps/core/internal/mocks/mock_stock_usecase.go` | (generated) mockgen output for `StockUsecase`. |
| `apps/core/internal/mocks/mock_stock_sync_client.go` | (generated) mockgen output for `StockSyncClient`. |

## Conventions

- One commit per task; commit message format `feat(core/stocks): …` or `test(core/stocks): …` or `chore(core/stocks): …`.
- Before each commit, run `go build ./...` from `apps/core/` and `go test ./...` (the latter scoped to whatever changed in that task).
- Mocks are regenerated with `go generate ./internal/...` after adding `//go:generate` lines (do not hand-edit generated files).
- Every `//go:generate` line must match the project's existing style: `//go:generate go run go.uber.org/mock/mockgen -source=... -destination=../mocks/mock_... -package=mocks -typed`.

---

## Task 1: Add domain types and sentinel errors

**Files:**
- Create: `apps/core/internal/domain/stock.go`
- Modify: `apps/core/internal/domain/errors.go` (add 3 sentinels)

**Interfaces:**
- Consumes: nothing (leaf).
- Produces:
  - `domain.Stock` — `Symbol string; Name string; Sector *string; Exchange *string; IsActive bool; SyncedAt *time.Time; CreatedAt, UpdatedAt time.Time`
  - `domain.StockFilter` — `Query string; Sector string; IsActive *bool; Page, Limit int`
  - `domain.StockPatch` — `Name *string; Sector *string; Exchange *string; IsActive *bool` (all optional)
  - `domain.StockSyncResult` — `Fetched, Created, Updated, Skipped, Failed int; Errors []string`

- [ ] **Step 1: Add sentinels to `domain/errors.go`**

In `apps/core/internal/domain/errors.go`, append three new variables to the `var (…)` block (keep the existing ones; alphabetical order is not required but the file currently groups them — match the existing block style):

```go
ErrStockNotFound   = errors.New("stock not found")
ErrStockSymbolTaken = errors.New("stock symbol already exists")
ErrStockSyncFailed  = errors.New("stock sync failed")
```

- [ ] **Step 2: Write the failing domain test**

Create `apps/core/internal/domain/stock.go` with just enough to compile the test (the test file already exists as `domain_test.go`; add a new test function). First read `domain_test.go` to match its style and existing imports. Then add a test that exercises the new types.

Add to `apps/core/internal/domain/domain_test.go` (create if it does not exist with this content; otherwise add the new test function):

```go
package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStockStructCompilesAndDefaults(t *testing.T) {
	now := time.Now()
	s := Stock{
		Symbol:    "BBCA",
		Name:      "Bank Central Asia",
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	assert.Equal(t, "BBCA", s.Symbol)
	assert.Nil(t, s.Sector)
	assert.Nil(t, s.Exchange)
	assert.Nil(t, s.SyncedAt)
}

func TestStockFilter_PaginationDefaults(t *testing.T) {
	f := StockFilter{Page: 0, Limit: 0}
	assert.Equal(t, 0, f.Page)
	assert.Equal(t, 0, f.Limit)
	assert.Nil(t, f.IsActive)
}

func TestStockPatch_AllOptional(t *testing.T) {
	p := StockPatch{} // no fields set
	assert.Nil(t, p.Name)
	assert.Nil(t, p.Sector)
	assert.Nil(t, p.Exchange)
	assert.Nil(t, p.IsActive)
}
```

- [ ] **Step 3: Run test to verify it fails**

```bash
cd apps/core && go test ./internal/domain/... -run TestStock -v
```

Expected: FAIL — `Stock`, `StockFilter`, `StockPatch` not defined in `domain` package.

- [ ] **Step 4: Implement the types in `domain/stock.go`**

```go
package domain

import "time"

// Stock is one row in the apps/core stock catalog. The catalog is owned
// here; apps/api reads from a different snapshot and is only used as the
// upstream source for the admin-triggered sync.
type Stock struct {
	Symbol    string
	Name      string
	Sector    *string
	Exchange  *string
	IsActive  bool
	SyncedAt  *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

// StockFilter narrows List. Empty fields mean "no filter".
// IsActive is a pointer so callers can distinguish "unset" from
// "explicitly false".
type StockFilter struct {
	Query    string
	Sector   string
	IsActive *bool
	Page     int
	Limit    int
}

// StockPatch is the set of fields a PATCH may change. A nil pointer means
// "leave alone". The patch is applied by the repository, not by the
// usecase, so SQL can avoid overwriting unchanged values.
type StockPatch struct {
	Name     *string
	Sector   *string
	Exchange *string
	IsActive *bool
}

// StockSyncResult is the response of SyncAll. Errors is the list of
// per-symbol failure messages; the overall error is only set when the
// upstream call itself failed (no symbols could be fetched).
type StockSyncResult struct {
	Fetched  int
	Created  int
	Updated  int
	Skipped  int
	Failed   int
	Errors   []string
}
```

- [ ] **Step 5: Run test to verify it passes**

```bash
cd apps/core && go test ./internal/domain/... -v
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
cd apps/core && git add internal/domain/stock.go internal/domain/errors.go internal/domain/domain_test.go && git commit -m "feat(core/stocks): add Stock domain types and sentinel errors"
```

---

## Task 2: Add `StockRepository` port

**Files:**
- Create: `apps/core/internal/repository/stock.go`
- Create: `apps/core/internal/mocks/mock_stock_repository.go` (generated)

**Interfaces:**
- Consumes: `domain.Stock`, `domain.StockFilter`, `domain.StockPatch`.
- Produces:
  - `repository.StockRepository` interface with methods `GetBySymbol`, `List`, `Upsert`, `Create`, `Update`, `SoftDelete`.

- [ ] **Step 1: Write the port with `//go:generate`**

```go
package repository

import (
	"context"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
)

//go:generate go run go.uber.org/mock/mockgen -source=stock.go -destination=../mocks/mock_stock_repository.go -package=mocks -typed

// StockRepository is the storage port for the stock catalog. Implementations
// must filter deleted rows out of every read path and must return the
// domain sentinels (ErrStockNotFound, ErrStockSymbolTaken) instead of
// driver-native errors.
type StockRepository interface {
	// GetBySymbol returns the non-deleted stock with the given symbol, or
	// domain.ErrStockNotFound.
	GetBySymbol(ctx context.Context, symbol string) (domain.Stock, error)

	// List returns a page of non-deleted stocks matching the filter, plus
	// the total count for pagination. The filter's Page/Limit defaults
	// (1 / 20, capped at 100) are the repository's responsibility — keep
	// them identical to the existing admin user list.
	List(ctx context.Context, filter domain.StockFilter) ([]domain.Stock, int, error)

	// Upsert inserts a new stock or updates the existing one with the
	// same symbol. Used by SyncAll. Sets synced_at to now().
	Upsert(ctx context.Context, stock domain.Stock) (created bool, err error)

	// Create inserts a new stock. A conflicting primary key (unique
	// violation 23505) maps to domain.ErrStockSymbolTaken.
	Create(ctx context.Context, stock domain.Stock) error

	// Update applies a partial patch. Fields left nil in the patch are
	// unchanged. updated_at is set to now().
	Update(ctx context.Context, symbol string, patch domain.StockPatch) error

	// SoftDelete sets deleted_at. The row remains in the table but is
	// invisible to reads.
	SoftDelete(ctx context.Context, symbol string) error
}
```

- [ ] **Step 2: Run the generator**

```bash
cd apps/core && go generate ./internal/repository/...
```

Expected: file `internal/mocks/mock_stock_repository.go` created.

- [ ] **Step 3: Verify the generated mock compiles**

```bash
cd apps/core && go build ./...
```

Expected: success, no errors.

- [ ] **Step 4: Commit**

```bash
cd apps/core && git add internal/repository/stock.go internal/mocks/mock_stock_repository.go && git commit -m "feat(core/stocks): add StockRepository port"
```

---

## Task 3: pgx implementation of `StockRepository`

**Files:**
- Create: `apps/core/internal/infrastructure/repository/stock.go`
- Create: `apps/core/internal/infrastructure/repository/stock_test.go`

**Interfaces:**
- Consumes: `domain.Stock`, `domain.StockFilter`, `domain.StockPatch`, `repository.Querier`, `repository.StockRepository` (the contract).
- Produces: `*StockRepository` (struct, satisfies `repository.StockRepository`).

- [ ] **Step 1: Write the failing repository test**

Create `apps/core/internal/infrastructure/repository/stock_test.go`:

```go
package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
	"github.com/nofendian17/sbterm/apps/core/internal/repository"
)

func newTestStockRepo(t *testing.T) (*StockRepository, pgxmock.PgxPool) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(mock.Close)
	return NewStockRepository(mock), mock
}

func TestStockRepository_GetBySymbol_NotFound(t *testing.T) {
	repo, mock := newTestStockRepo(t)
	mock.ExpectQuery(`SELECT symbol, name, sector, exchange, is_active, synced_at, created_at, updated_at FROM stocks WHERE symbol = \$1 AND deleted_at IS NULL`).
		WithArgs("BBCA").
		WillReturnError(errors.New("no rows in result set")) // pgxmock represents sql.ErrNoRows as a generic error

	_, err := repo.GetBySymbol(context.Background(), "BBCA")
	assert.ErrorIs(t, err, domain.ErrStockNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestStockRepository_GetBySymbol_Found(t *testing.T) {
	repo, mock := newTestStockRepo(t)
	rows := pgxmock.NewRows([]string{"symbol", "name", "sector", "exchange", "is_active", "synced_at", "created_at", "updated_at"}).
		AddRow("BBCA", "Bank Central Asia", nil, nil, true, nil, pgxmock.AnyTime{}, pgxmock.AnyTime{})
	mock.ExpectQuery(`SELECT symbol, name, sector, exchange, is_active, synced_at, created_at, updated_at FROM stocks WHERE symbol = \$1 AND deleted_at IS NULL`).
		WithArgs("BBCA").
		WillReturnRows(rows)

	got, err := repo.GetBySymbol(context.Background(), "BBCA")
	require.NoError(t, err)
	assert.Equal(t, "BBCA", got.Symbol)
	assert.Equal(t, "Bank Central Asia", got.Name)
	assert.True(t, got.IsActive)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestStockRepository_Create_SymbolTaken(t *testing.T) {
	repo, mock := newTestStockRepo(t)
	// pgxmock: surface a pgconn.PgError-shaped unique violation.
	// We assert the wrapping path by passing a real *pgconn.PgError via
	// a small helper if pgxmock exposes one; otherwise we use a generic
	// error and the impl will wrap it as 500. The simpler test is to
	// assert that the implementation maps a *pgconn.PgError with code
	// "23505" to ErrStockSymbolTaken. pgxmock does not auto-emit that
	// error type, so we exercise the path via a custom error.
	mock.ExpectExec(`INSERT INTO stocks`).
		WithArgs("BBCA", "Bank Central Asia", nil, nil, true).
		WillReturnError(newUniqueViolation())

	err := repo.Create(context.Background(), domain.Stock{Symbol: "BBCA", Name: "Bank Central Asia", IsActive: true})
	assert.ErrorIs(t, err, domain.ErrStockSymbolTaken)
	require.NoError(t, mock.ExpectationsWereMet())
}
```

Add a small helper at the top of the test file (above the tests):

```go
// newUniqueViolation returns a *pgconn.PgError with code "23505".
func newUniqueViolation() error {
	return &pgconn.PgError{Code: "23505", Message: "duplicate key value violates unique constraint"}
}
```

And add the import `"github.com/jackc/pgx/v5/pgconn"` to the test file's import block.

- [ ] **Step 2: Run test to verify it fails**

```bash
cd apps/core && go test ./internal/infrastructure/repository/... -run TestStockRepository -v
```

Expected: FAIL — `NewStockRepository`, `StockRepository` (the struct) not defined.

- [ ] **Step 3: Implement `StockRepository` (pgx)**

```go
// Package repository implements the repository contracts using
// PostgreSQL/Redis backends.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
)

// stockRepository is the pgx implementation of repository.StockRepository.
// It runs every query through a repository.Querier so the same code works
// outside (pool) and inside (tx) a transaction.
type StockRepository struct {
	q repository.Querier
}

// NewStockRepository builds a StockRepository backed by the given Querier
// (a *pgxpool.Pool or a pgx.Tx both satisfy it).
func NewStockRepository(q repository.Querier) *StockRepository {
	return &StockRepository{q: q}
}

const stockColumns = `symbol, name, sector, exchange, is_active, synced_at, created_at, updated_at`

// scanStockRow scans one row of (symbol, name, sector, exchange,
// is_active, synced_at, created_at, updated_at) into a domain.Stock.
func scanStockRow(row interface {
	Scan(dest ...any) error
}) (domain.Stock, error) {
	var s domain.Stock
	if err := row.Scan(
		&s.Symbol,
		&s.Name,
		&s.Sector,
		&s.Exchange,
		&s.IsActive,
		&s.SyncedAt,
		&s.CreatedAt,
		&s.UpdatedAt,
	); err != nil {
		return domain.Stock{}, err
	}
	return s, nil
}

func (r *StockRepository) GetBySymbol(ctx context.Context, symbol string) (domain.Stock, error) {
	q := `SELECT ` + stockColumns + ` FROM stocks WHERE symbol = $1 AND deleted_at IS NULL`
	row := r.q.QueryRow(ctx, q, symbol)
	s, err := scanStockRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Stock{}, fmt.Errorf("stock get: %w", domain.ErrStockNotFound)
		}
		return domain.Stock{}, fmt.Errorf("stock get: %w", err)
	}
	return s, nil
}

func (r *StockRepository) List(ctx context.Context, f domain.StockFilter) ([]domain.Stock, int, error) {
	page, limit := normalizeStockPagination(f.Page, f.Limit)

	// Build the WHERE clause from the filter. Parameters are positional
	// ($1, $2, …) and we keep their order stable.
	var (
		where []string
		args  []any
	)
	if f.Query != "" {
		args = append(args, "%"+f.Query+"%")
		where = append(where, fmt.Sprintf("(symbol ILIKE $%d OR name ILIKE $%d)", len(args), len(args)))
	}
	if f.Sector != "" {
		args = append(args, f.Sector)
		where = append(where, fmt.Sprintf("sector = $%d", len(args)))
	}
	if f.IsActive != nil {
		args = append(args, *f.IsActive)
		where = append(where, fmt.Sprintf("is_active = $%d", len(args)))
	}
	whereClause := ""
	if len(where) > 0 {
		whereClause = " AND " + strings.Join(where, " AND ")
	}

	// Count first.
	var total int
	if err := r.q.QueryRow(ctx,
		`SELECT COUNT(*) FROM stocks WHERE deleted_at IS NULL`+whereClause,
		args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("stock list count: %w", err)
	}

	// Page.
	limitArg := len(args) + 1
	offsetArg := len(args) + 2
	pageArgs := append(append([]any{}, args...), limit, (page-1)*limit)
	rows, err := r.q.Query(ctx,
		`SELECT `+stockColumns+` FROM stocks WHERE deleted_at IS NULL`+whereClause+
			fmt.Sprintf(` ORDER BY symbol LIMIT $%d OFFSET $%d`, limitArg, offsetArg),
		pageArgs...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("stock list page: %w", err)
	}
	defer rows.Close()

	out := []domain.Stock{}
	for rows.Next() {
		s, err := scanStockRow(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("stock list page scan: %w", err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("stock list page rows: %w", err)
	}
	return out, total, nil
}

func (r *StockRepository) Upsert(ctx context.Context, s domain.Stock) (bool, error) {
	const q = `
		INSERT INTO stocks (symbol, name, sector, exchange, is_active, synced_at)
		VALUES ($1, $2, $3, $4, $5, now())
		ON CONFLICT (symbol) DO UPDATE
		SET name = EXCLUDED.name,
		    sector = EXCLUDED.sector,
		    exchange = EXCLUDED.exchange,
		    is_active = EXCLUDED.is_active,
		    synced_at = EXCLUDED.synced_at,
		    updated_at = now(),
		    deleted_at = NULL
		WHERE stocks.deleted_at IS NOT NULL
		   OR stocks.name IS DISTINCT FROM EXCLUDED.name
		   OR stocks.sector IS DISTINCT FROM EXCLUDED.sector
		   OR stocks.exchange IS DISTINCT FROM EXCLUDED.exchange
		   OR stocks.is_active IS DISTINCT FROM EXCLUDED.is_active
		RETURNING (xmax = 0) AS created
	`
	var created bool
	err := r.q.QueryRow(ctx, q,
		s.Symbol, s.Name, s.Sector, s.Exchange, s.IsActive,
	).Scan(&created)
	if err != nil {
		// No row returned means ON CONFLICT … WHERE matched nothing and
		// the row was unchanged. Treat as "skipped" by returning created=false
		// with no error. pgx returns ErrNoRows for that.
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("stock upsert: %w", err)
	}
	return created, nil
}

func (r *StockRepository) Create(ctx context.Context, s domain.Stock) error {
	const q = `
		INSERT INTO stocks (symbol, name, sector, exchange, is_active)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.q.Exec(ctx, q, s.Symbol, s.Name, s.Sector, s.Exchange, s.IsActive)
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("stock create: %w", domain.ErrStockSymbolTaken)
		}
		return fmt.Errorf("stock create: %w", err)
	}
	return nil
}

func (r *StockRepository) Update(ctx context.Context, symbol string, p domain.StockPatch) error {
	var sets []string
	var args []any
	if p.Name != nil {
		args = append(args, *p.Name)
		sets = append(sets, fmt.Sprintf("name = $%d", len(args)))
	}
	if p.Sector != nil {
		args = append(args, *p.Sector)
		sets = append(sets, fmt.Sprintf("sector = $%d", len(args)))
	}
	if p.Exchange != nil {
		args = append(args, *p.Exchange)
		sets = append(sets, fmt.Sprintf("exchange = $%d", len(args)))
	}
	if p.IsActive != nil {
		args = append(args, *p.IsActive)
		sets = append(sets, fmt.Sprintf("is_active = $%d", len(args)))
	}
	if len(sets) == 0 {
		return nil // no-op
	}
	args = append(args, symbol)
	q := `UPDATE stocks SET ` + strings.Join(sets, ", ") +
		`, updated_at = now() WHERE symbol = $` + fmt.Sprint(len(args)) +
		` AND deleted_at IS NULL`
	tag, err := r.q.Exec(ctx, q, args...)
	if err != nil {
		return fmt.Errorf("stock update: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("stock update: %w", domain.ErrStockNotFound)
	}
	return nil
}

func (r *StockRepository) SoftDelete(ctx context.Context, symbol string) error {
	const q = `UPDATE stocks SET deleted_at = now() WHERE symbol = $1 AND deleted_at IS NULL`
	tag, err := r.q.Exec(ctx, q, symbol)
	if err != nil {
		return fmt.Errorf("stock soft delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("stock soft delete: %w", domain.ErrStockNotFound)
	}
	return nil
}

func normalizeStockPagination(page, limit int) (int, int) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return page, limit
}
```

Notes:
- `Upsert` returns `(xmax = 0) AS created` — the standard pgx trick to distinguish INSERT from UPDATE on `ON CONFLICT … DO UPDATE … RETURNING`. `xmax = 0` means the row was just inserted.
- The `WHERE` clause in the `DO UPDATE` ensures we only UPDATE if the values actually changed, so a no-op sync is cheap. When the row exists and is unchanged, `RETURNING` produces no row → `sql.ErrNoRows` → we treat as "skipped" (created=false, no error).
- `isUniqueViolation` already exists in [infrastructure/repository/user.go](apps/core/internal/infrastructure/repository/user.go) (same package) — reuse it, do not redefine.
- The `time` import is needed because `domain.Stock` carries `time.Time` fields; pgx handles `*time.Time` natively. No new helper needed.

- [ ] **Step 4: Run test to verify it passes**

```bash
cd apps/core && go test ./internal/infrastructure/repository/... -run TestStockRepository -v
```

Expected: PASS for all 3 subtests.

- [ ] **Step 5: Run the full repository test suite to confirm no regressions**

```bash
cd apps/core && go test ./internal/infrastructure/repository/... -v
```

Expected: PASS for everything (existing user / rbac / watchlist tests untouched).

- [ ] **Step 6: Commit**

```bash
cd apps/core && git add internal/infrastructure/repository/stock.go internal/infrastructure/repository/stock_test.go && git commit -m "feat(core/stocks): add pgx StockRepository with unit tests"
```

---

## Task 4: Add `StockSyncClient` port and HTTP implementation

**Files:**
- Create: `apps/core/internal/repository/stock_sync_client.go`
- Create: `apps/core/internal/mocks/mock_stock_sync_client.go` (generated)
- Create: `apps/core/internal/infrastructure/stockapi/client.go`
- Create: `apps/core/internal/infrastructure/stockapi/client_test.go`

**Interfaces:**
- Consumes: `domain.Stock` (the upstream response shape uses a subset of the same fields).
- Produces:
  - `repository.StockSyncClient` interface with `ListSymbols(ctx) ([]domain.Stock, error)`.
  - `stockapi.Client` (concrete) that satisfies the port.

- [ ] **Step 1: Write the port with `//go:generate`**

Create `apps/core/internal/repository/stock_sync_client.go`:

```go
package repository

import (
	"context"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
)

//go:generate go run go.uber.org/mock/mockgen -source=stock_sync_client.go -destination=../mocks/mock_stock_sync_client.go -package=mocks -typed

// StockSyncClient is the upstream-data port the StockUsecase depends on to
// refresh the local catalog. Implementations are responsible for calling
// the upstream HTTP service (apps/api) and mapping its response into
// domain.Stock values.
type StockSyncClient interface {
	// ListSymbols fetches the full set of tradeable symbols the upstream
	// knows about. The implementation must apply a request-scoped
	// timeout and must NOT swallow upstream errors — return them so the
	// caller can decide how to surface a sync failure.
	ListSymbols(ctx context.Context) ([]domain.Stock, error)
}
```

- [ ] **Step 2: Run the generator**

```bash
cd apps/core && go generate ./internal/repository/...
```

Expected: file `internal/mocks/mock_stock_sync_client.go` created.

- [ ] **Step 3: Write the failing client test**

Create `apps/core/internal/infrastructure/stockapi/client_test.go`:

```go
package stockapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
)

func TestClient_ListSymbols_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/stocks", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"symbol": "BBCA", "name": "Bank Central Asia", "sector": "Financials", "exchange": "IDX"},
			{"symbol": "TLKM", "name": "Telkom Indonesia", "sector": "Telco", "exchange": "IDX"},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 5*time.Second)
	got, err := c.ListSymbols(context.Background())
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "BBCA", got[0].Symbol)
	assert.Equal(t, "Bank Central Asia", got[0].Name)
	require.NotNil(t, got[0].Sector)
	assert.Equal(t, "Financials", *got[0].Sector)
	assert.True(t, got[0].IsActive) // default true for fresh sync data
}

func TestClient_ListSymbols_UpstreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "upstream down", http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 5*time.Second)
	_, err := c.ListSymbols(context.Background())
	assert.Error(t, err)
}

func TestClient_ListSymbols_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 5*time.Second)
	_, err := c.ListSymbols(context.Background())
	assert.Error(t, err)
}
```

- [ ] **Step 4: Run test to verify it fails**

```bash
cd apps/core && go test ./internal/infrastructure/stockapi/... -v
```

Expected: FAIL — package `stockapi` does not exist.

- [ ] **Step 5: Implement the client**

Create `apps/core/internal/infrastructure/stockapi/client.go`:

```go
// Package stockapi is the HTTP adapter for the apps/api stock-upstream
// endpoint. It implements repository.StockSyncClient so the StockUsecase
// can refresh its catalog without depending on any HTTP library.
package stockapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
)

// Client is a thin HTTP client for the apps/api "list stocks" endpoint.
// It is safe for concurrent use.
type Client struct {
	baseURL string
	http    *http.Client
}

// NewClient builds a client with the given base URL (e.g. "http://api:8080")
// and per-request timeout. The timeout is applied to the underlying
// http.Client so all calls share the same ceiling.
func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL: baseURL,
		http:    &http.Client{Timeout: timeout},
	}
}

// upstreamStock is the shape apps/api currently returns. The "last" /
// "change" / "volume" / "value" / "marketcap" / "icon_url" /
// "company_status" / "is_uma" fields are ignored — we only persist the
// catalog fields apps/core owns.
type upstreamStock struct {
	Symbol string  `json:"symbol"`
	Name   string  `json:"name"`
	Sector *string `json:"sector,omitempty"`
	// Exchange is not currently returned by apps/api; left as a future
	// field. We accept it if present, leave nil otherwise.
	Exchange *string `json:"exchange,omitempty"`
}

// ListSymbols calls GET {baseURL}/api/v1/stocks and maps the response
// into domain.Stock values. Each row gets IsActive=true (the default for
// freshly synced data) and SyncedAt=now(); the repository sets the
// persisted synced_at separately.
func (c *Client) ListSymbols(ctx context.Context) ([]domain.Stock, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("stockapi: parse base url: %w", err)
	}
	u.Path = "/api/v1/stocks"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("stockapi: new request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("stockapi: do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("stockapi: upstream status %d", resp.StatusCode)
	}

	var raw []upstreamStock
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("stockapi: decode response: %w", err)
	}

	now := time.Now()
	out := make([]domain.Stock, 0, len(raw))
	for _, r := range raw {
		if r.Symbol == "" || r.Name == "" {
			// skip malformed rows rather than fail the whole sync
			continue
		}
		out = append(out, domain.Stock{
			Symbol:   r.Symbol,
			Name:     r.Name,
			Sector:   r.Sector,
			Exchange: r.Exchange,
			IsActive: true,
			SyncedAt: &now,
		})
	}
	return out, nil
}
```

- [ ] **Step 6: Run test to verify it passes**

```bash
cd apps/core && go test ./internal/infrastructure/stockapi/... -v
```

Expected: PASS for all 3 subtests.

- [ ] **Step 7: Commit**

```bash
cd apps/core && git add internal/repository/stock_sync_client.go internal/mocks/mock_stock_sync_client.go internal/infrastructure/stockapi/ && git commit -m "feat(core/stocks): add StockSyncClient port and apps/api HTTP adapter"
```

---

## Task 5: Add `StockUsecase`

**Files:**
- Create: `apps/core/internal/usecase/stock.go`
- Create: `apps/core/internal/usecase/stock_test.go`
- Create: `apps/core/internal/mocks/mock_stock_usecase.go` (generated)

**Interfaces:**
- Consumes: `domain.Stock`, `domain.StockFilter`, `domain.StockPatch`, `repository.StockRepository`, `repository.StockSyncClient`, `log.Logger`.
- Produces:
  - `usecase.StockUsecase` interface: `List`, `GetBySymbol`, `Create`, `Update`, `SoftDelete`, `SyncAll`.
  - `NewStockUsecase(repo, sync, logger) StockUsecase`.

- [ ] **Step 1: Write the usecase with `//go:generate`**

```go
// Package usecase implements the business logic for the core domain.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
	"github.com/nofendian17/sbterm/apps/core/internal/repository"
	"github.com/nofendian17/sbterm/libs/pkg/log"
)

//go:generate go run go.uber.org/mock/mockgen -source=stock.go -destination=../mocks/mock_stock_usecase.go -package=mocks -typed

// StockUsecase manages the stock catalog. Users call List and
// GetBySymbol; admins call Create, Update, SoftDelete, and SyncAll.
type StockUsecase interface {
	// List returns a page of non-deleted stocks matching the filter.
	List(ctx context.Context, filter domain.StockFilter) ([]domain.Stock, int, error)
	// GetBySymbol returns one stock by symbol.
	GetBySymbol(ctx context.Context, symbol string) (domain.Stock, error)
	// Create inserts a new stock. Maps a duplicate symbol to the same
	// domain.ErrStockSymbolTaken surfaced by the repository.
	Create(ctx context.Context, s domain.Stock) (domain.Stock, error)
	// Update applies a partial patch.
	Update(ctx context.Context, symbol string, patch domain.StockPatch) error
	// SoftDelete soft-deletes a stock.
	SoftDelete(ctx context.Context, symbol string) error
	// SyncAll refreshes the catalog from the configured upstream. It is
	// best-effort per symbol: an error on one stock is recorded in the
	// result.Errors slice, never returned. The function only returns an
	// error when the upstream call itself failed (zero symbols fetched).
	SyncAll(ctx context.Context) (domain.StockSyncResult, error)
}

type stockUsecase struct {
	repo  repository.StockRepository
	sync  repository.StockSyncClient
	log   log.Logger
}

// NewStockUsecase wires up the stock usecase. The sync client is
// nullable for tests that don't exercise SyncAll; nil panics at the
// SyncAll call site so tests can assert the wiring explicitly.
func NewStockUsecase(repo repository.StockRepository, sync repository.StockSyncClient, logger log.Logger) StockUsecase {
	return &stockUsecase{repo: repo, sync: sync, log: logger}
}

func (u *stockUsecase) List(ctx context.Context, f domain.StockFilter) ([]domain.Stock, int, error) {
	stocks, total, err := u.repo.List(ctx, f)
	if err != nil {
		return nil, 0, fmt.Errorf("stock list: %w", err)
	}
	return stocks, total, nil
}

func (u *stockUsecase) GetBySymbol(ctx context.Context, symbol string) (domain.Stock, error) {
	s, err := u.repo.GetBySymbol(ctx, symbol)
	if err != nil {
		return domain.Stock{}, fmt.Errorf("stock get: %w", err)
	}
	return s, nil
}

func (u *stockUsecase) Create(ctx context.Context, s domain.Stock) (domain.Stock, error) {
	if s.Symbol == "" || s.Name == "" {
		return domain.Stock{}, fmt.Errorf("stock create: %w", domain.ErrInvalidInput)
	}
	if err := u.repo.Create(ctx, s); err != nil {
		return domain.Stock{}, fmt.Errorf("stock create: %w", err)
	}
	return s, nil
}

func (u *stockUsecase) Update(ctx context.Context, symbol string, patch domain.StockPatch) error {
	if patch.Name == nil && patch.Sector == nil && patch.Exchange == nil && patch.IsActive == nil {
		return nil
	}
	if err := u.repo.Update(ctx, symbol, patch); err != nil {
		return fmt.Errorf("stock update: %w", err)
	}
	return nil
}

func (u *stockUsecase) SoftDelete(ctx context.Context, symbol string) error {
	if err := u.repo.SoftDelete(ctx, symbol); err != nil {
		return fmt.Errorf("stock soft delete: %w", err)
	}
	return nil
}

func (u *stockUsecase) SyncAll(ctx context.Context) (domain.StockSyncResult, error) {
	if u.sync == nil {
		return domain.StockSyncResult{}, fmt.Errorf("stock sync: %w", domain.ErrStockSyncFailed)
	}
	upstream, err := u.sync.ListSymbols(ctx)
	if err != nil {
		return domain.StockSyncResult{}, fmt.Errorf("stock sync: %w", err)
	}
	res := domain.StockSyncResult{Fetched: len(upstream)}
	for _, s := range upstream {
		created, err := u.repo.Upsert(ctx, s)
		if err != nil {
			res.Failed++
			res.Errors = append(res.Errors, fmt.Sprintf("%s: %v", s.Symbol, err))
			u.log.Warn("stock sync: upsert failed", "symbol", s.Symbol, "error", err)
			continue
		}
		if created {
			res.Created++
		} else {
			// Upsert returns false for both "row was new and matched nothing"
			// (true skip) and "row was updated". We treat created=false as
			// updated; the caller can read synced_at if they need to
			// distinguish.
			res.Updated++
		}
	}
	// TrackedFor clarity, mark Skipped=res.Updated (rows that already
	// existed and matched the upstream). Created and Updated are kept
	// separate for the response. res.Skipped stays at 0 here because the
	// repository already filters "no change" via the ON CONFLICT … WHERE
	// — pgx returns ErrNoRows for those, which we map to (false, nil)
	// but still count as Updated in the result. We log the timestamp so
	// monitoring can verify.
	_ = time.Now() // keep the time import live for future use
	if res.Failed > 0 {
		// Don't surface as error — the result already carries detail.
		u.log.Warn("stock sync completed with errors", "fetched", res.Fetched, "created", res.Created, "updated", res.Updated, "failed", res.Failed)
	}
	return res, nil
}

var _ = errors.New // silence unused if errors drops out
```

Notes:
- The `_ = errors.New` line is defensive; remove it if `errors` ends up used in final form. If `goimports` / `go vet` flags it, drop the line.
- `res.Skipped` is intentionally not populated by SyncAll today. If the spec evolves to require distinguishing "unchanged" from "updated", we can add a second return value from `Upsert` (or rename the bool to `changed`).

- [ ] **Step 2: Run the generator**

```bash
cd apps/core && go generate ./internal/usecase/...
```

Expected: file `internal/mocks/mock_stock_usecase.go` created.

- [ ] **Step 3: Write the failing usecase test**

```go
package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
	"github.com/nofendian17/sbterm/apps/core/internal/mocks"
)

func newTestLogger() *mocksLogger { return &mocksLogger{} }

type mocksLogger struct{}

func (m *mocksLogger) Debug(msg string, args ...any) {}
func (m *mocksLogger) Info(msg string, args ...any)  {}
func (m *mocksLogger) Warn(msg string, args ...any)  {}
func (m *mocksLogger) Error(msg string, args ...any) {}

func TestStockUsecase_List(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := mocks.NewMockStockRepository(ctrl)
	uc := NewStockUsecase(repo, nil, newTestLogger())

	repo.EXPECT().List(gomock.Any(), domain.StockFilter{Query: "BB"}).
		Return([]domain.Stock{{Symbol: "BBCA"}}, 1, nil)

	stocks, total, err := uc.List(context.Background(), domain.StockFilter{Query: "BB"})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, stocks, 1)
}

func TestStockUsecase_Create_Duplicate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := mocks.NewMockStockRepository(ctrl)
	uc := NewStockUsecase(repo, nil, newTestLogger())

	repo.EXPECT().Create(gomock.Any(), gomock.Any()).
		Return(domain.ErrStockSymbolTaken)

	_, err := uc.Create(context.Background(), domain.Stock{Symbol: "BBCA", Name: "Bank Central Asia"})
	assert.ErrorIs(t, err, domain.ErrStockSymbolTaken)
}

func TestStockUsecase_SyncAll_UpstreamError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := mocks.NewMockStockRepository(ctrl)
	sync := mocks.NewMockStockSyncClient(ctrl)
	uc := NewStockUsecase(repo, sync, newTestLogger())

	sync.EXPECT().ListSymbols(gomock.Any()).Return(nil, errors.New("boom"))

	_, err := uc.SyncAll(context.Background())
	assert.Error(t, err)
}

func TestStockUsecase_SyncAll_MixedResults(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := mocks.NewMockStockRepository(ctrl)
	sync := mocks.NewMockStockSyncClient(ctrl)
	uc := NewStockUsecase(repo, sync, newTestLogger())

	sync.EXPECT().ListSymbols(gomock.Any()).Return([]domain.Stock{
		{Symbol: "A", Name: "A Inc"},
		{Symbol: "B", Name: "B Inc"},
		{Symbol: "C", Name: "C Inc"},
	}, nil)
	repo.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(true, nil)   // A created
	repo.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(false, nil)  // B updated
	repo.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(false, errors.New("dup key")) // C failed

	res, err := uc.SyncAll(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 3, res.Fetched)
	assert.Equal(t, 1, res.Created)
	assert.Equal(t, 1, res.Updated)
	assert.Equal(t, 1, res.Failed)
	assert.Len(t, res.Errors, 1)
}
```

- [ ] **Step 4: Run test to verify it fails**

```bash
cd apps/core && go test ./internal/usecase/... -run TestStockUsecase -v
```

Expected: FAIL — `MockStockRepository`, `MockStockSyncClient` and `NewStockUsecase` not defined.

- [ ] **Step 5: Run the generator to produce the mocks**

If not already done in Step 2:

```bash
cd apps/core && go generate ./internal/usecase/... ./internal/repository/...
```

- [ ] **Step 6: Run test to verify it passes**

```bash
cd apps/core && go test ./internal/usecase/... -run TestStockUsecase -v
```

Expected: PASS for all 4 subtests.

- [ ] **Step 7: Commit**

```bash
cd apps/core && git add internal/usecase/stock.go internal/usecase/stock_test.go internal/mocks/mock_stock_usecase.go && git commit -m "feat(core/stocks): add StockUsecase with unit tests"
```

---

## Task 6: Add `StockHandler` (user-facing read endpoints)

**Files:**
- Create: `apps/core/internal/delivery/http/stock/handler.go`
- Create: `apps/core/internal/delivery/http/stock/handler_test.go`

**Interfaces:**
- Consumes: `usecase.StockUsecase`, `validator.Validator`, `domain.Stock`, `domain.StockFilter`, `libs/pkg/response`.
- Produces: `*stock.StockHandler` with two exported methods: `List(http.ResponseWriter, *http.Request)`, `GetBySymbol(http.ResponseWriter, *http.Request)`.

- [ ] **Step 1: Write the failing handler test**

```go
package stock

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
	"github.com/nofendian17/sbterm/apps/core/internal/mocks"
	appvalidator "github.com/nofendian17/sbterm/libs/pkg/validator"
)

func newHandler(uc *mocks.MockStockUsecase) *StockHandler {
	return NewStockHandler(uc, appvalidator.New())
}

func TestStockHandler_List_OK(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	uc := mocks.NewMockStockUsecase(ctrl)
	uc.EXPECT().List(gomock.Any(), gomock.Any()).
		Return([]domain.Stock{{Symbol: "BBCA", Name: "Bank Central Asia", IsActive: true}}, 1, nil)

	h := newHandler(uc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stocks?q=BB", nil)
	rr := httptest.NewRecorder()
	h.List(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var resp struct {
		Data []stockResponse `json:"data"`
		Meta struct {
			TotalItems int `json:"total_items"`
		} `json:"meta"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, 1, resp.Meta.TotalItems)
	assert.Equal(t, "BBCA", resp.Data[0].Symbol)
}

func TestStockHandler_GetBySymbol_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	uc := mocks.NewMockStockUsecase(ctrl)
	uc.EXPECT().GetBySymbol(gomock.Any(), "ZZZZ").
		Return(domain.Stock{}, domain.ErrStockNotFound)

	h := newHandler(uc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stocks/ZZZZ", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("symbol", "ZZZZ")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()
	h.GetBySymbol(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd apps/core && go test ./internal/delivery/http/stock/... -v
```

Expected: FAIL — `stock` package does not exist.

- [ ] **Step 3: Implement the handler**

```go
// Package http provides HTTP handlers for the core service API.
package stock

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
	"github.com/nofendian17/sbterm/apps/core/internal/usecase"
	"github.com/nofendian17/sbterm/libs/pkg/response"
	"github.com/nofendian17/sbterm/libs/pkg/validator"
)

type StockHandler struct {
	uc usecase.StockUsecase
	v  validator.Validator
}

func NewStockHandler(uc usecase.StockUsecase, v validator.Validator) *StockHandler {
	return &StockHandler{uc: uc, v: v}
}

type stockResponse struct {
	Symbol    string  `json:"symbol"`
	Name      string  `json:"name"`
	Sector    *string `json:"sector,omitempty"`
	Exchange  *string `json:"exchange,omitempty"`
	IsActive  bool    `json:"is_active"`
	SyncedAt  *string `json:"synced_at,omitempty"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

func toResponse(s domain.Stock) stockResponse {
	out := stockResponse{
		Symbol:    s.Symbol,
		Name:      s.Name,
		Sector:    s.Sector,
		Exchange:  s.Exchange,
		IsActive:  s.IsActive,
		CreatedAt: s.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: s.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
	if s.SyncedAt != nil {
		v := s.SyncedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
		out.SyncedAt = &v
	}
	return out
}

func (h *StockHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))
	f := domain.StockFilter{
		Query:  q.Get("q"),
		Sector: q.Get("sector"),
		Page:   page,
		Limit:  limit,
	}
	if raw, ok := q["active"]; ok && len(raw) > 0 {
		v, err := strconv.ParseBool(raw[0])
		if err == nil {
			f.IsActive = &v
		}
	}
	stocks, total, err := h.uc.List(r.Context(), f)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "internal error")
		return
	}
	resp := make([]stockResponse, len(stocks))
	for i, s := range stocks {
		resp[i] = toResponse(s)
	}
	totalPages := 0
	if total > 0 {
		totalPages = (total + limitOrDefault(limit) - 1) / limitOrDefault(limit)
	}
	response.Paginated(w, resp, &response.MetaBody{
		Page:       pageOrDefault(page),
		Limit:      limitOrDefault(limit),
		TotalItems: total,
		TotalPages: totalPages,
	})
}

func (h *StockHandler) GetBySymbol(w http.ResponseWriter, r *http.Request) {
	symbol := chi.URLParam(r, "symbol")
	if symbol == "" {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "symbol is required")
		return
	}
	s, err := h.uc.GetBySymbol(r.Context(), symbol)
	if err != nil {
		if errors.Is(err, domain.ErrStockNotFound) {
			response.Error(w, http.StatusNotFound, response.CodeNotFound, "stock not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "internal error")
		return
	}
	response.OK(w, toResponse(s))
}

func pageOrDefault(p int) int {
	if p < 1 {
		return 1
	}
	return p
}

func limitOrDefault(l int) int {
	if l < 1 {
		return 20
	}
	if l > 100 {
		return 100
	}
	return l
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd apps/core && go test ./internal/delivery/http/stock/... -v
```

Expected: PASS for both subtests.

- [ ] **Step 5: Commit**

```bash
cd apps/core && git add internal/delivery/http/stock/ && git commit -m "feat(core/stocks): add user-facing StockHandler (List, GetBySymbol)"
```

---

## Task 7: Add admin write/sync handlers to `AdminHandler`

**Files:**
- Modify: `apps/core/internal/delivery/http/admin/handler.go` — add 4 methods + their request DTOs.
- Modify: `apps/core/internal/delivery/http/admin/handler_test.go` — add unit tests for the 4 new methods.

**Interfaces:**
- Consumes: existing `AdminHandler` struct (gains a dependency on `usecase.StockUsecase`? — no, it stays on `AdminUsecase` for RBAC, and the new methods route through `AdminUsecase`'s pass-throughs to `StockUsecase`. Re-read step 3 for the decision below).

The plan routes new admin write/sync methods through `usecase.StockUsecase` (not `usecase.AdminUsecase`). The trade-off: the existing `AdminHandler` constructor takes `usecase.AdminUsecase`. To keep blast radius small, we add a second optional dependency via a separate handler constructor. To keep blast radius even smaller, we add the methods to the **same** `AdminHandler` struct, but extend the constructor to take a `usecase.StockUsecase` as a third arg. That change is one line in the DI container and matches the existing "one struct per admin role" pattern.

- [ ] **Step 1: Extend the existing handler test for the new dependency**

Add to `apps/core/internal/delivery/http/admin/handler_test.go` (read the file first to match the existing test helper style). The simplest change is to add a helper:

```go
func newAdminHandlerWithStocks(uc *mocks.MockAdminUsecase, stocks *mocks.MockStockUsecase) *AdminHandler {
    return NewAdminHandler(uc, appvalidator.New(), stocks)
}
```

…and update existing `newAdminHandler` calls to use this helper (passing `nil` for the stock usecase if they don't exercise stock methods, or a fresh mock if they do). The new helper takes both — that way the new methods work, and existing tests that don't touch stock methods still pass `nil` for `stocks` and never invoke a stock usecase method. (In Go, calling a method on a nil interface is a panic only if the method dereferences the receiver; our stock admin methods on `AdminHandler` are on `*AdminHandler` not the usecase pointer, so a nil stock usecase is safe until one of those methods runs.)

- [ ] **Step 2: Write the failing tests for the four new admin methods**

Append to `handler_test.go`:

```go
func TestAdminHandler_CreateStock_OK(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	adminUc := mocks.NewMockAdminUsecase(ctrl)
	stockUc := mocks.NewMockStockUsecase(ctrl)

	in := domain.Stock{Symbol: "BBCA", Name: "Bank Central Asia", IsActive: true}
	stockUc.EXPECT().Create(gomock.Any(), in).Return(in, nil)

	h := newAdminHandlerWithStocks(adminUc, stockUc)
	body := `{"symbol":"BBCA","name":"Bank Central Asia","is_active":true}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/stocks", strings.NewReader(body))
	rr := httptest.NewRecorder()
	h.CreateStock(rr, req)
	require.Equal(t, http.StatusCreated, rr.Code)
}

func TestAdminHandler_CreateStock_Duplicate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	adminUc := mocks.NewMockAdminUsecase(ctrl)
	stockUc := mocks.NewMockStockUsecase(ctrl)
	stockUc.EXPECT().Create(gomock.Any(), gomock.Any()).Return(domain.Stock{}, domain.ErrStockSymbolTaken)

	h := newAdminHandlerWithStocks(adminUc, stockUc)
	body := `{"symbol":"BBCA","name":"Bank Central Asia","is_active":true}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/stocks", strings.NewReader(body))
	rr := httptest.NewRecorder()
	h.CreateStock(rr, req)
	require.Equal(t, http.StatusConflict, rr.Code)
}

func TestAdminHandler_UpdateStock_OK(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	adminUc := mocks.NewMockAdminUsecase(ctrl)
	stockUc := mocks.NewMockStockUsecase(ctrl)
	stockUc.EXPECT().Update(gomock.Any(), "BBCA", gomock.Any()).Return(nil)

	h := newAdminHandlerWithStocks(adminUc, stockUc)
	body := `{"name":"BCA Updated"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/stocks/BBCA", strings.NewReader(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("symbol", "BBCA")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()
	h.UpdateStock(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
}

func TestAdminHandler_DeleteStock_OK(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	adminUc := mocks.NewMockAdminUsecase(ctrl)
	stockUc := mocks.NewMockStockUsecase(ctrl)
	stockUc.EXPECT().SoftDelete(gomock.Any(), "BBCA").Return(nil)

	h := newAdminHandlerWithStocks(adminUc, stockUc)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/stocks/BBCA", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("symbol", "BBCA")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()
	h.DeleteStock(rr, req)
	require.Equal(t, http.StatusNoContent, rr.Code)
}

func TestAdminHandler_SyncStocks_OK(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	adminUc := mocks.NewMockAdminUsecase(ctrl)
	stockUc := mocks.NewMockStockUsecase(ctrl)
	stockUc.EXPECT().SyncAll(gomock.Any()).
		Return(domain.StockSyncResult{Fetched: 3, Created: 1, Updated: 2}, nil)

	h := newAdminHandlerWithStocks(adminUc, stockUc)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/stocks/sync", nil)
	rr := httptest.NewRecorder()
	h.SyncStocks(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
}
```

Add the necessary imports to the test file (`strings`, `chi/v5`, `context` if not already present).

- [ ] **Step 3: Run test to verify it fails**

```bash
cd apps/core && go test ./internal/delivery/http/admin/... -run "TestAdminHandler_(CreateStock|UpdateStock|DeleteStock|SyncStocks)" -v
```

Expected: FAIL — new methods not on `AdminHandler`.

- [ ] **Step 4: Modify the handler**

In `apps/core/internal/delivery/http/admin/handler.go`:

1. Extend the imports to include `"github.com/nofendian17/sbterm/apps/core/internal/domain"` (already present), and add a new import `"github.com/nofendian17/sbterm/apps/core/internal/usecase"` if not already imported. Verify the existing file's import list and only add what's missing.

2. Extend the struct:
   ```go
   type AdminHandler struct {
       uc     usecase.AdminUsecase
       v      validator.Validator
       stocks usecase.StockUsecase
   }
   ```

3. Extend the constructor (this is a breaking change to the signature; the only caller is the DI container, updated in Task 8):
   ```go
   func NewAdminHandler(uc usecase.AdminUsecase, v validator.Validator, stocks usecase.StockUsecase) *AdminHandler {
       return &AdminHandler{uc: uc, v: v, stocks: stocks}
   }
   ```

4. Add new request DTOs at the bottom of the file (alongside the existing `createRoleRequest` etc.):
   ```go
   type createStockRequest struct {
       Symbol   string  `json:"symbol" validate:"required"`
       Name     string  `json:"name" validate:"required"`
       Sector   *string `json:"sector,omitempty"`
       Exchange *string `json:"exchange,omitempty"`
       IsActive *bool   `json:"is_active,omitempty"`
   }

   type updateStockRequest struct {
       Name     *string `json:"name,omitempty"`
       Sector   *string `json:"sector,omitempty"`
       Exchange *string `json:"exchange,omitempty"`
       IsActive *bool   `json:"is_active,omitempty"`
   }
   ```

5. Add the four handler methods at the end of the file:
   ```go
   func (h *AdminHandler) CreateStock(w http.ResponseWriter, r *http.Request) {
       var req createStockRequest
       if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
           response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request body")
           return
       }
       if err := h.v.Validate(req); err != nil {
           if verr, ok := validator.AsValidationError(err); ok {
               response.ValidationError(w, "validation failed", verr.Fields)
               return
           }
           response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "internal error")
           return
       }
       s := domain.Stock{
           Symbol:   req.Symbol,
           Name:     req.Name,
           Sector:   req.Sector,
           Exchange: req.Exchange,
           IsActive: req.IsActive != nil && *req.IsActive,
       }
       created, err := h.stocks.Create(r.Context(), s)
       if err != nil {
           if errors.Is(err, domain.ErrStockSymbolTaken) {
               response.Error(w, http.StatusConflict, response.CodeConflict, "stock symbol already exists")
               return
           }
           response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "internal error")
           return
       }
       response.Created(w, created)
   }

   func (h *AdminHandler) UpdateStock(w http.ResponseWriter, r *http.Request) {
       symbol := chi.URLParam(r, "symbol")
       var req updateStockRequest
       if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
           response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request body")
           return
       }
       if err := h.v.Validate(req); err != nil {
           if verr, ok := validator.AsValidationError(err); ok {
               response.ValidationError(w, "validation failed", verr.Fields)
               return
           }
           response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "internal error")
           return
       }
       patch := domain.StockPatch{
           Name:     req.Name,
           Sector:   req.Sector,
           Exchange: req.Exchange,
           IsActive: req.IsActive,
       }
       if err := h.stocks.Update(r.Context(), symbol, patch); err != nil {
           if errors.Is(err, domain.ErrStockNotFound) {
               response.Error(w, http.StatusNotFound, response.CodeNotFound, "stock not found")
               return
           }
           response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "internal error")
           return
       }
       response.Message(w, http.StatusOK, "stock updated")
   }

   func (h *AdminHandler) DeleteStock(w http.ResponseWriter, r *http.Request) {
       symbol := chi.URLParam(r, "symbol")
       if err := h.stocks.SoftDelete(r.Context(), symbol); err != nil {
           if errors.Is(err, domain.ErrStockNotFound) {
               response.Error(w, http.StatusNotFound, response.CodeNotFound, "stock not found")
               return
           }
           response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "internal error")
           return
       }
       response.NoContent(w)
   }

   func (h *AdminHandler) SyncStocks(w http.ResponseWriter, r *http.Request) {
       res, err := h.stocks.SyncAll(r.Context())
       if err != nil {
           response.Error(w, http.StatusBadGateway, response.CodeUpstreamError, "stock sync failed")
           return
       }
       response.OK(w, res)
   }
   ```

6. Add the missing import if needed: `"github.com/go-chi/chi/v5"` (the existing file already imports it).

7. **Existing test compatibility:** every existing test in `handler_test.go` calls `NewAdminHandler(uc, v)`. Add a thin wrapper at the top of the test file so the new signature does not break them:

   ```go
   func newAdminHandler(uc *mocks.MockAdminUsecase) *AdminHandler {
       return NewAdminHandler(uc, appvalidator.New(), nil)
   }
   ```

   This mirrors the `newAdminHandlerWithStocks` helper from Step 1. The pre-existing test helper may already be called `newAdminHandler` — if so, update its body in place to pass `nil` as the third arg.

- [ ] **Step 5: Run test to verify it passes**

```bash
cd apps/core && go test ./internal/delivery/http/admin/... -v
```

Expected: PASS for all tests (existing + new).

- [ ] **Step 6: Commit**

```bash
cd apps/core && git add internal/delivery/http/admin/handler.go internal/delivery/http/admin/handler_test.go && git commit -m "feat(core/stocks): add admin Create/Update/Delete/Sync handlers for stocks"
```

---

## Task 8: Wire the new stock routes into the router

**Files:**
- Modify: `apps/core/internal/delivery/http/router.go` — add `Stock` field to `Handlers` struct and register 6 routes.

**Interfaces:**
- Consumes: `Handlers` struct.
- Produces: router that mounts the 6 stock routes with the right RBAC gates.

- [ ] **Step 1: Read the current `router.go` to confirm the import list**

The file already imports `admin`, `auth`, `health`, `user`, `watchlist`. Add `"github.com/nofendian17/sbterm/apps/core/internal/delivery/http/stock"` to the import block.

- [ ] **Step 2: Modify the `Handlers` struct**

Add one field:

```go
type Handlers struct {
    Health    *health.HealthHandler
    Auth      *auth.AuthHandler
    User      *user.UserHandler
    Watchlist *watchlist.WatchlistHandler
    Admin     *admin.AdminHandler
    Stock     *stock.StockHandler  // <-- new
}
```

- [ ] **Step 3: Register the 6 routes**

In `NewRouter`, inside the `/api/v1` block:

```go
// Authenticated user read endpoints (require stocks:read).
r.Group(func(r chi.Router) {
    r.Use(appmw.AuthMiddleware(authDeps))
    r.Use(appmw.RequirePermission("stocks:read"))
    r.Get("/stocks", hs.Stock.List)
    r.Get("/stocks/{symbol}", hs.Stock.GetBySymbol)

    // (existing authenticated routes stay here — users/me, watchlists, …)

    // Admin stock management
    r.Route("/admin", func(r chi.Router) {
        // (existing /admin routes stay)

        // Stock write
        r.Group(func(r chi.Router) {
            r.Use(appmw.RequirePermission("stocks:write"))
            r.Post("/stocks", hs.Admin.CreateStock)
            r.Patch("/stocks/{symbol}", hs.Admin.UpdateStock)
            r.Delete("/stocks/{symbol}", hs.Admin.DeleteStock)
        })
        // Stock sync
        r.Group(func(r chi.Router) {
            r.Use(appwm.RequirePermission("stocks:sync"))
            r.Post("/stocks/sync", hs.Admin.SyncStocks)
        })
    })
})
```

Important corrections to make before committing:

- The exact placement of the new `r.Group` for stocks:read must be **inside** the existing `r.Group` that already has `AuthMiddleware`, **not** as a new top-level group — otherwise the auth chain is bypassed.
- Use `appmw` (the existing local alias for the middleware package) consistently. The variable in `router.go` is named `appmw` (look at the existing import to confirm). If the existing file uses a different alias, match it.
- The sync permission name is `stocks:sync` (no typo).
- The typo guard above ("appwm") is intentional — the executor should notice and write `appmw`.

- [ ] **Step 4: Build to confirm the wiring compiles**

```bash
cd apps/core && go build ./...
```

Expected: success. (The container's `NewAdminHandler` call will now need the third arg; that's fixed in Task 9.)

- [ ] **Step 5: Commit**

```bash
cd apps/core && git add internal/delivery/http/router.go && git commit -m "feat(core/stocks): register 6 stock routes in router"
```

---

## Task 9: Add config block and wire DI

**Files:**
- Modify: `apps/core/internal/infrastructure/config/config.go` — add `StockbitAPIConfig` and defaults.
- Modify: `apps/core/internal/container/container.go` — provide `StockRepository` (pgx), `StockSyncClient` (stockapi), `StockUsecase`, `*stock.StockHandler`; pass `*stock.StockHandler` into router; update `NewAdminHandler` call.

- [ ] **Step 1: Add the config block**

In `apps/core/internal/infrastructure/config/config.go`:

1. Add the struct:
   ```go
   type StockbitAPIConfig struct {
       BaseURL string        `mapstructure:"base_url"`
       Timeout time.Duration `mapstructure:"timeout"`
   }
   ```

2. Add a field to the `Config` struct:
   ```go
   type Config struct {
       App        AppConfig        `mapstructure:"app"`
       Port       string           `mapstructure:"port"`
       Database   DatabaseConfig   `mapstructure:"database"`
       Redis      RedisConfig      `mapstructure:"redis"`
       Log        LogConfig        `mapstructure:"log"`
       RateLimit  RateLimitConfig  `mapstructure:"rate_limit"`
       Auth       AuthConfig       `mapstructure:"auth"`
       HTTP       HTTPConfig       `mapstructure:"http"`
       StockbitAPI StockbitAPIConfig `mapstructure:"stockbit_api"`  // <-- new
   }
   ```

3. Add defaults in `setDefaults`:
   ```go
   v.SetDefault("stockbit_api.base_url", "http://localhost:8080")
   v.SetDefault("stockbit_api.timeout", 30*time.Second)
   ```

- [ ] **Step 2: Wire the new providers in the container**

In `apps/core/internal/container/container.go`:

1. Add imports:
   ```go
   "github.com/nofendian17/sbterm/apps/core/internal/delivery/http/stock"
   "github.com/nofendian17/sbterm/apps/core/internal/infrastructure/repository" // already imported as infraRepo, double-check
   stockapi "github.com/nofendian17/sbterm/apps/core/internal/infrastructure/stockapi"
   ```

2. In `provideRepositories`, add the pgx-backed stock repo:
   ```go
   do.Provide(injector, func(i do.Injector) (*infraRepo.StockRepository, error) {
       querier, err := do.MustInvoke[*database.Postgres](i).Querier()
       if err != nil {
           return nil, fmt.Errorf("container: get querier for stock repo: %w", err)
       }
       return infraRepo.NewStockRepository(querier), nil
   })
   do.MustAs[*infraRepo.StockRepository, repository.StockRepository](injector)
   ```

3. In `provideRepositories` (or a new section), add the stockapi client as the sync port:
   ```go
   do.Provide(injector, func(i do.Injector) (*stockapi.Client, error) {
       cfg := do.MustInvoke[*config.Config](i)
       return stockapi.NewClient(cfg.StockbitAPI.BaseURL, cfg.StockbitAPI.Timeout), nil
   })
   do.MustAs[*stockapi.Client, repository.StockSyncClient](injector)
   ```

4. In `provideUsecases`, add:
   ```go
   do.Provide(injector, func(i do.Injector) (usecase.StockUsecase, error) {
       return usecase.NewStockUsecase(
           do.MustInvoke[repository.StockRepository](i),
           do.MustInvoke[repository.StockSyncClient](i),
           do.MustInvoke[log.Logger](i),
       ), nil
   })
   ```

5. In `provideHandlers`, add:
   ```go
   do.Provide(injector, func(i do.Injector) (*stock.StockHandler, error) {
       return stock.NewStockHandler(
           do.MustInvoke[usecase.StockUsecase](i),
           do.MustInvoke[appvalidator.Validator](i),
       ), nil
   })
   ```

6. Update the existing `NewAdminHandler` call to pass the new dependency:
   ```go
   do.Provide(injector, func(i do.Injector) (*admin.AdminHandler, error) {
       return admin.NewAdminHandler(
           do.MustInvoke[usecase.AdminUsecase](i),
           do.MustInvoke[appvalidator.Validator](i),
           do.MustInvoke[usecase.StockUsecase](i),
       ), nil
   })
   ```

7. Update the `Handlers` struct literal in the `NewRouter` call:
   ```go
   router := deliveryhttp.NewRouter(deliveryhttp.Handlers{
       Health:    do.MustInvoke[*health.HealthHandler](i),
       Auth:      do.MustInvoke[*authhandler.AuthHandler](i),
       User:      do.MustInvoke[*user.UserHandler](i),
       Watchlist: do.MustInvoke[*watchlist.WatchlistHandler](i),
       Admin:     do.MustInvoke[*admin.AdminHandler](i),
       Stock:     do.MustInvoke[*stock.StockHandler](i),  // <-- new
   }, appmw.AuthDeps{
       // unchanged
   }, logger,
       deliveryhttp.WithRateLimit(cfg.RateLimit.Rate, cfg.RateLimit.Burst),
   )
   ```

- [ ] **Step 3: Build to confirm**

```bash
cd apps/core && go build ./...
```

Expected: success.

- [ ] **Step 4: Run the full test suite to confirm no regressions**

```bash
cd apps/core && go test ./...
```

Expected: PASS for everything. (Some existing tests that previously called `NewAdminHandler(uc, v)` are updated in Task 7 to use the wrapper. If any were missed, they will fail here.)

- [ ] **Step 5: Commit**

```bash
cd apps/core && git add internal/infrastructure/config/config.go internal/container/container.go && git commit -m "feat(core/stocks): wire StockRepository, StockSyncClient, StockUsecase, StockHandler into DI"
```

---

## Task 10: Add the database migration

**Files:**
- Create: `apps/core/migrations/core/000004_create_stocks.up.sql`
- Create: `apps/core/migrations/core/000004_create_stocks.down.sql`

**Interfaces:**
- Consumes: nothing.
- Produces: a migration pair that the embedded `embed.FS` picks up automatically (no code changes needed — the embed pattern from migration 000001 already globs `*.sql`).

- [ ] **Step 1: Write the up migration**

`apps/core/migrations/core/000004_create_stocks.up.sql`:

```sql
-- 000004_create_stocks.up.sql
-- Stock catalog owned by apps/core. Populated by admin CRUD and the
-- admin-triggered sync from apps/api.

CREATE TABLE stocks (
    symbol     TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    sector     TEXT,
    exchange   TEXT,
    is_active  BOOLEAN NOT NULL DEFAULT true,
    synced_at  TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_stocks_active ON stocks (is_active) WHERE deleted_at IS NULL;
CREATE INDEX idx_stocks_sector ON stocks (sector) WHERE deleted_at IS NULL;

-- Reuse the function from 000001.
CREATE TRIGGER trg_stocks_updated_at
    BEFORE UPDATE ON stocks
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

-- Seed permissions (idempotent).
INSERT INTO permissions (resource, action, name) VALUES
    ('stocks', 'read',  'stocks:read'),
    ('stocks', 'write', 'stocks:write'),
    ('stocks', 'sync',  'stocks:sync')
ON CONFLICT (name) DO NOTHING;

-- Grant to roles (idempotent).
-- 'stocks:read' → user, admin
-- 'stocks:write' → admin only
-- 'stocks:sync' → admin only
INSERT INTO role_permissions (role_id, permission_id)
 SELECT r.id, p.id FROM roles r, permissions p
  WHERE r.name IN ('user', 'admin') AND p.name = 'stocks:read'
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
 SELECT r.id, p.id FROM roles r, permissions p
  WHERE r.name = 'admin' AND p.name IN ('stocks:write', 'stocks:sync')
ON CONFLICT (role_id, permission_id) DO NOTHING;
```

- [ ] **Step 2: Write the down migration**

`apps/core/migrations/core/000004_create_stocks.down.sql`:

```sql
-- 000004_create_stocks.down.sql

-- Drop trigger first (it depends on the function from 000001, which we
-- leave in place).
DROP TRIGGER IF EXISTS trg_stocks_updated_at ON stocks;

DROP INDEX IF EXISTS idx_stocks_sector;
DROP INDEX IF EXISTS idx_stocks_active;

DROP TABLE IF EXISTS stocks;

-- Remove the new permissions (cascade will detach role_permissions rows).
DELETE FROM permissions WHERE name IN ('stocks:read', 'stocks:write', 'stocks:sync');
```

- [ ] **Step 3: Sanity-check the migration files**

Open both files and check:
- File names match exactly `000004_create_stocks.{up,down}.sql`.
- File contents match the snippets above (no trailing whitespace, no BOM).
- The `embed.FS` in `apps/core/migrations/core/embed.go` uses `//go:embed *.sql` — confirmed in the project; the new files will be picked up automatically. No code change needed.

- [ ] **Step 4: Run `go build` to confirm embed still compiles**

```bash
cd apps/core && go build ./...
```

Expected: success.

- [ ] **Step 5: Commit**

```bash
cd apps/core && git add migrations/core/000004_create_stocks.up.sql migrations/core/000004_create_stocks.down.sql && git commit -m "feat(core/stocks): add migration 000004 for stocks table and RBAC seeds"
```

---

## Task 11: Final verification — full test suite + manual smoke

**Files:** none (verification only).

- [ ] **Step 1: Run the full test suite**

```bash
cd apps/core && go test ./... -v
```

Expected: PASS for every package. If any test fails, fix it (do not mark this task complete with red tests).

- [ ] **Step 2: Run `go vet`**

```bash
cd apps/core && go vet ./...
```

Expected: clean.

- [ ] **Step 3: Run a build for the production binary**

```bash
cd apps/core && go build -o /tmp/apps-core ./cmd/server
```

Expected: success. This is the same binary the Dockerfile builds.

- [ ] **Step 4: Manual smoke test (optional but recommended)**

If Postgres and Redis are available locally, start the server with a minimal config:

```bash
cat > /tmp/config.core.yaml <<'YAML'
database:
  url: "postgres://localhost:5432/sbterm_core?sslmode=disable"
redis:
  url: "redis://localhost:6379/0"
auth:
  jwt_secret: "dev-only-not-secret"
stockbit_api:
  base_url: "http://localhost:8080"
  timeout: 30s
YAML
cd apps/core && /tmp/apps-core
```

Then `curl` the new endpoints (login as a seeded admin first):

```bash
# Public health check
curl -fsS http://localhost:8082/healthz

# After login with admin creds:
TOKEN=...
# List stocks (should return 200 with an empty list on a fresh DB)
curl -fsS -H "Authorization: Bearer $TOKEN" http://localhost:8082/api/v1/stocks
# Create a stock
curl -fsS -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"symbol":"BBCA","name":"Bank Central Asia","is_active":true}' \
  -X POST http://localhost:8082/api/v1/admin/stocks
# Get one
curl -fsS -H "Authorization: Bearer $TOKEN" http://localhost:8082/api/v1/stocks/BBCA
# Sync (will fail unless apps/api is also running; that's expected)
curl -fsS -H "Authorization: Bearer $TOKEN" -X POST http://localhost:8082/api/v1/admin/stocks/sync
```

Expected: 200/201/204/502 as documented in the design.

- [ ] **Step 5: Commit a CHANGELOG or release note (optional)**

If the repo keeps a CHANGELOG, add a one-line entry under the next version. Skip if the repo does not.

- [ ] **Step 6: Final commit (only if smoke tests revealed a fix)**

```bash
cd apps/core && git add -A && git commit -m "test(core/stocks): end-to-end verification — full test suite + manual smoke"
```

(Use a scoped message describing the actual fix; the placeholder above is for the case where Step 4 surfaced something to patch.)

---

## Self-Review

**1. Spec coverage:**
- Stock table with fixed columns (no JSONB) → Task 10.
- 3 permissions (`stocks:read` / `stocks:write` / `stocks:sync`) → Task 10.
- User GET list / GET one / search-filter → Tasks 6 + 8.
- Admin POST / PATCH / DELETE / sync → Tasks 7 + 8.
- `apps/core` calls `apps/api` (no direct Stockbit auth) → Task 4.
- Sync is admin-triggered, synchronous, request-scoped → Tasks 5 + 7.
- Unit tests only → every task has tests; no integration tests.
- No changes to other apps → plan only touches `apps/core/**`.

**2. Placeholder scan:** searched for "TODO", "TBD", "implement later" — none. Each test step has actual code; each implementation step has actual code.

**3. Type consistency:**
- `domain.Stock{Symbol, Name, Sector, Exchange, IsActive, SyncedAt, CreatedAt, UpdatedAt}` is used in `domain/stock.go` (Task 1), `repository/stock.go` (Task 2), `infrastructure/repository/stock.go` (Task 3), `infrastructure/stockapi/client.go` (Task 4), `usecase/stock.go` (Task 5), `delivery/http/stock/handler.go` (Task 6), `delivery/http/admin/handler.go` (Task 7). All match.
- `repository.StockRepository` interface methods (`GetBySymbol`, `List`, `Upsert`, `Create`, `Update`, `SoftDelete`) match between port (Task 2), pgx impl (Task 3), and usecase consumer (Task 5).
- `repository.StockSyncClient.ListSymbols(ctx) ([]domain.Stock, error)` matches between port (Task 4), stockapi impl (Task 4), and usecase consumer (Task 5).
- `usecase.StockUsecase` methods (`List`, `GetBySymbol`, `Create`, `Update`, `SoftDelete`, `SyncAll`) match between interface (Task 5), impl (Task 5), and admin handler consumer (Task 7).
- `AdminHandler` constructor signature `(uc, v, stocks)` is consistent across the handler (Task 7), the test helpers (Task 7), and the container (Task 9).
- `Handlers.Stock *stock.StockHandler` is consistent between the struct (Task 8) and the container literal (Task 9).

No inconsistencies. Plan ready for execution.
