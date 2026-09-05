# Sector CRUD Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Give `apps/core` full HTTP CRUD for the `sectors` master table (today read-only via `GetByName`, writes by hand-SQL).

**Architecture:** Vertical slice inside `apps/core` following the stock pattern: extend `repository.SectorRepository` port → pgx impl in `infrastructure/repository/sector.go` → new `usecase.SectorUsecase` → new `delivery/http/sector` handler → wire in `container` + `router`. Reuses existing `stocks:read` / `stocks:write` permissions (same precedent as `000006` company profiles, which rode on stocks perms with no new seeds) — no migration needed.

**Tech Stack:** Go 1.26, chi/v5, pgx/v5, samber/do/v2, mockgen, pgxmock, testify, validator.

**Spec:** User request 2026-09-05 ("buatkan CRUD sector lengkap"); context: `docs/superpowers/specs/2026-09-05-stocks-feature-design.md` (sector normalization) and `apps/core/migrations/core/000003_create_sectors.up.sql` (table shape: `id UUID PK`, `name TEXT UNIQUE`, `created_at/updated_at/deleted_at`).

## Global Constraints

- Go module is `github.com/nofendian17/sbterm/apps/core` (do not introduce new top-level modules).
- `apps/core/internal/repository/Querier` is the only DB seam — every new repository method must take a `Querier`, never `*pgxpool.Pool` or `pgx.Tx` directly.
- All read paths filter `deleted_at IS NULL` (soft delete, consistent with `users` / `watchlists` / `stocks`).
- `libs/pkg` is the only cross-module import for utility packages (`log`, `response`, `validator`).
- No changes to `apps/api`, `apps/ws`, `apps/ingest`, `apps/stream`.
- `apps/core`'s port `:8082` and base URL `/api/v1` stay.
- All new tests are unit tests; use `pgxmock` for the pgx repository, `testify` + `mockgen` for the rest. No live Postgres / Redis calls in tests.
- Reuse `stocks:read` (list/get) and `stocks:write` (create/update/delete) — no new migration, no new RBAC seeds.

## File Structure

| File | Responsibility |
|---|---|
| `apps/core/internal/domain/errors.go` | + `ErrSectorNameTaken`, `ErrSectorHasStocks` sentinels |
| `apps/core/internal/repository/sector.go` | Extend port: `List`, `GetByID`, `Create`, `Update`, `SoftDelete` |
| `apps/core/internal/infrastructure/repository/sector.go` | pgx impl of the extended port |
| `apps/core/internal/infrastructure/repository/sector_test.go` | NEW pgxmock tests |
| `apps/core/internal/usecase/sector.go` | NEW `SectorUsecase` (trim/validate, sentinel passthrough) |
| `apps/core/internal/usecase/sector_test.go` | NEW mock tests |
| `apps/core/internal/delivery/http/sector/handler.go` | NEW `SectorHandler` (List/GetByID/Create/Update/Delete) |
| `apps/core/internal/delivery/http/sector/handler_test.go` | NEW handler tests |
| `apps/core/internal/delivery/http/router.go` | + 5 routes (`GET /sectors`, `GET /sectors/{id}`, `POST/PATCH/DELETE /admin/sectors...`) |
| `apps/core/internal/container/container.go` | Provide `SectorUsecase` + `SectorHandler`, add to `Handlers` |
| `apps/core/internal/mocks/` | Regenerate via `go generate` |

---

### Task 1: Port + sentinels

**Files:**
- Modify: `apps/core/internal/domain/errors.go`
- Modify: `apps/core/internal/repository/sector.go`

**Interfaces:**
- Consumes: existing `domain.Sector{ID, Name}`, `domain.ErrSectorNotFound`, `domain.ErrInvalidInput`
- Produces: `ErrSectorNameTaken`, `ErrSectorHasStocks`; extended `SectorRepository` interface (used by Tasks 2–3)

- [ ] **Step 1: Add sentinels**

In `apps/core/internal/domain/errors.go`, after the `ErrSectorNotFound` line, add:

```go
ErrSectorNameTaken = errors.New("sector name already exists")
ErrSectorHasStocks = errors.New("sector has stocks")
```

- [ ] **Step 2: Extend the port**

Replace `apps/core/internal/repository/sector.go` interface block with:

```go
// SectorRepository manages the manually-curated sectors master table.
// Every read filters soft-deleted rows.
type SectorRepository interface {
	// List returns all non-deleted sectors ordered by name.
	List(ctx context.Context) ([]domain.Sector, error)

	// GetByID returns one non-deleted sector, or domain.ErrSectorNotFound.
	GetByID(ctx context.Context, id string) (domain.Sector, error)

	// GetByName returns the non-deleted sector with the given name, or
	// domain.ErrSectorNotFound.
	GetByName(ctx context.Context, name string) (domain.Sector, error)

	// Create inserts a sector. A name conflict (23505) maps to
	// domain.ErrSectorNameTaken.
	Create(ctx context.Context, name string) (domain.Sector, error)

	// Update renames a sector. Returns domain.ErrSectorNotFound when missing,
	// domain.ErrSectorNameTaken on name conflict.
	Update(ctx context.Context, id, name string) error

	// SoftDelete sets deleted_at. Refused with domain.ErrSectorHasStocks
	// when live stocks still reference the sector (stocks.sector_id FK only
	// blocks hard deletes); domain.ErrSectorNotFound when missing.
	SoftDelete(ctx context.Context, id string) error
}
```

Keep the `//go:generate` line and imports (`context`, `domain`) unchanged.

- [ ] **Step 3: Regenerate mocks, verify build**

```bash
cd apps/core && go generate ./internal/repository/... && go build ./...
```

Expected: PASS (mocks gain the new methods; `StockUsecase` still compiles since it depends on the same interface).

- [ ] **Step 4: Commit**

```bash
git add apps/core/internal/domain/errors.go apps/core/internal/repository/sector.go apps/core/internal/mocks/mock_sector_repository.go
git commit -m "feat(core/sectors): extend port with CRUD and add sentinels"
```

---

### Task 2: pgx repository + tests

**Files:**
- Modify: `apps/core/internal/infrastructure/repository/sector.go`
- Create: `apps/core/internal/infrastructure/repository/sector_test.go`
- Test: `apps/core/internal/infrastructure/repository/sector_test.go`

**Interfaces:**
- Consumes: extended `contract.SectorRepository` from Task 1; existing helpers `isUniqueViolation` (user.go), `isPgErrorCode` + `foreignKeyViolationCode`/`uniqueViolationCode` (watchlist.go/user.go)
- Produces: working CRUD impl used by Task 3

- [ ] **Step 1: Write the failing test**

Create `apps/core/internal/infrastructure/repository/sector_test.go`:

```go
package repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/pashagolub/pgxmock/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
)

func newTestSectorRepo(t *testing.T) (*SectorRepository, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(mock.Close)
	return NewSectorRepository(AdaptQuerier(mock)), mock
}

func TestSectorRepository_Create_NameTaken(t *testing.T) {
	repo, mock := newTestSectorRepo(t)
	mock.ExpectQuery(`INSERT INTO sectors`).
		WithArgs("Financials").
		WillReturnError(pgUniqueViolation())

	_, err := repo.Create(context.Background(), "Financials")
	assert.ErrorIs(t, err, domain.ErrSectorNameTaken)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSectorRepository_SoftDelete_HasStocks(t *testing.T) {
	repo, mock := newTestSectorRepo(t)
	mock.ExpectExec(`UPDATE sectors SET deleted_at`).
		WithArgs("s1").
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs("s1").
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))

	err := repo.SoftDelete(context.Background(), "s1")
	assert.ErrorIs(t, err, domain.ErrSectorHasStocks)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSectorRepository_GetByID_NotFound(t *testing.T) {
	repo, mock := newTestSectorRepo(t)
	mock.ExpectQuery(`FROM sectors WHERE id = \$1 AND deleted_at IS NULL`).
		WithArgs("s1").
		WillReturnError(sql.ErrNoRows)

	_, err := repo.GetByID(context.Background(), "s1")
	assert.ErrorIs(t, err, domain.ErrSectorNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd apps/core && go test ./internal/infrastructure/repository/ -run TestSectorRepository -v
```

Expected: FAIL — `undefined: NewSectorRepository` mismatch is fine, but specifically `Create`, `SoftDelete`, `GetByID` methods do not exist.

- [ ] **Step 3: Write minimal implementation**

In `apps/core/internal/infrastructure/repository/sector.go`, keep `NewSectorRepository` and `GetByName`, add:

```go
// List returns all non-deleted sectors ordered by name.
func (r *SectorRepository) List(ctx context.Context) ([]domain.Sector, error) {
	rows, err := r.q.Query(ctx,
		`SELECT id, name FROM sectors WHERE deleted_at IS NULL ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("sector list: %w", err)
	}
	defer rows.Close()

	out := []domain.Sector{}
	for rows.Next() {
		var s domain.Sector
		if err := rows.Scan(&s.ID, &s.Name); err != nil {
			return nil, fmt.Errorf("sector list scan: %w", err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sector list rows: %w", err)
	}
	return out, nil
}

// GetByID returns one non-deleted sector.
func (r *SectorRepository) GetByID(ctx context.Context, id string) (domain.Sector, error) {
	var s domain.Sector
	err := r.q.QueryRow(ctx,
		`SELECT id, name FROM sectors WHERE id = $1 AND deleted_at IS NULL`, id,
	).Scan(&s.ID, &s.Name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Sector{}, fmt.Errorf("sector get: %w", domain.ErrSectorNotFound)
		}
		return domain.Sector{}, fmt.Errorf("sector get: %w", err)
	}
	return s, nil
}

// Create inserts a sector and returns it.
func (r *SectorRepository) Create(ctx context.Context, name string) (domain.Sector, error) {
	var s domain.Sector
	err := r.q.QueryRow(ctx,
		`INSERT INTO sectors (name) VALUES ($1) RETURNING id, name`, name,
	).Scan(&s.ID, &s.Name)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.Sector{}, fmt.Errorf("sector create: %w", domain.ErrSectorNameTaken)
		}
		return domain.Sector{}, fmt.Errorf("sector create: %w", err)
	}
	return s, nil
}

// Update renames a sector.
func (r *SectorRepository) Update(ctx context.Context, id, name string) error {
	tag, err := r.q.Exec(ctx,
		`UPDATE sectors SET name = $2, updated_at = now() WHERE id = $1 AND deleted_at IS NULL`,
		id, name)
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("sector update: %w", domain.ErrSectorNameTaken)
		}
		return fmt.Errorf("sector update: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("sector update: %w", domain.ErrSectorNotFound)
	}
	return nil
}

// SoftDelete marks the sector deleted. It refuses when live stocks still
// reference it, mirroring StockRepository.SoftDelete's watchlist guard.
func (r *SectorRepository) SoftDelete(ctx context.Context, id string) error {
	tag, err := r.q.Exec(ctx,
		`UPDATE sectors SET deleted_at = now()
		 WHERE id = $1 AND deleted_at IS NULL
		 AND NOT EXISTS (
			SELECT 1 FROM stocks
			WHERE stocks.sector_id = sectors.id
			AND stocks.deleted_at IS NULL
		 )`, id)
	if err != nil {
		return fmt.Errorf("sector soft delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		var exists bool
		if qerr := r.q.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM stocks WHERE sector_id = $1 AND deleted_at IS NULL)`,
			id,
		).Scan(&exists); qerr == nil && exists {
			return fmt.Errorf("sector soft delete: %w", domain.ErrSectorHasStocks)
		}
		return fmt.Errorf("sector soft delete: %w", domain.ErrSectorNotFound)
	}
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd apps/core && go test ./internal/infrastructure/repository/ -run TestSectorRepository -v
```

Expected: PASS (3 tests).

- [ ] **Step 5: Commit**

```bash
git add apps/core/internal/infrastructure/repository/sector.go apps/core/internal/infrastructure/repository/sector_test.go
git commit -m "feat(core/sectors): pgx CRUD with has-stocks guard"
```

---

### Task 3: Usecase + tests

**Files:**
- Create: `apps/core/internal/usecase/sector.go`
- Create: `apps/core/internal/usecase/sector_test.go`
- Test: `apps/core/internal/usecase/sector_test.go`

**Interfaces:**
- Consumes: `repository.SectorRepository` (Task 1), `domain.ErrInvalidInput`
- Produces: `SectorUsecase` interface + `NewSectorUsecase` (consumed by Task 4 container/handler)

- [ ] **Step 1: Write the failing test**

Create `apps/core/internal/usecase/sector_test.go`:

```go
package usecase

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
	"github.com/nofendian17/sbterm/apps/core/internal/mocks"
)

func TestSectorUsecase_Create_TrimsAndRejectsEmpty(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := mocks.NewMockSectorRepository(ctrl)
	uc := NewSectorUsecase(repo)

	_, err := uc.Create(context.Background(), "   ")
	assert.ErrorIs(t, err, domain.ErrInvalidInput)

	repo.EXPECT().Create(gomock.Any(), "Financials").
		Return(domain.Sector{ID: "s1", Name: "Financials"}, nil)
	got, err := uc.Create(context.Background(), "  Financials ")
	require.NoError(t, err)
	assert.Equal(t, "s1", got.ID)
}

func TestSectorUsecase_SoftDelete_BlockedByStocks(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := mocks.NewMockSectorRepository(ctrl)
	uc := NewSectorUsecase(repo)

	repo.EXPECT().SoftDelete(gomock.Any(), "s1").
		Return(domain.ErrSectorHasStocks)
	err := uc.SoftDelete(context.Background(), "s1")
	assert.ErrorIs(t, err, domain.ErrSectorHasStocks)
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd apps/core && go test ./internal/usecase/ -run TestSectorUsecase -v
```

Expected: FAIL with "undefined: NewSectorUsecase".

- [ ] **Step 3: Write minimal implementation**

Create `apps/core/internal/usecase/sector.go`:

```go
// Package usecase implements the business logic for the core domain.

package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
	"github.com/nofendian17/sbterm/apps/core/internal/repository"
)

//go:generate go run go.uber.org/mock/mockgen -source=sector.go -destination=../mocks/mock_sector_usecase.go -package=mocks -typed

// SectorUsecase manages the sectors master table. Reads are user-facing;
// writes are admin-gated by the router.
type SectorUsecase interface {
	List(ctx context.Context) ([]domain.Sector, error)
	GetByID(ctx context.Context, id string) (domain.Sector, error)
	Create(ctx context.Context, name string) (domain.Sector, error)
	Update(ctx context.Context, id, name string) error
	SoftDelete(ctx context.Context, id string) error
}

type sectorUsecase struct {
	repo repository.SectorRepository
}

// NewSectorUsecase wires up the sector usecase.
func NewSectorUsecase(repo repository.SectorRepository) SectorUsecase {
	return &sectorUsecase{repo: repo}
}

func (u *sectorUsecase) List(ctx context.Context) ([]domain.Sector, error) {
	sectors, err := u.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("sector list: %w", err)
	}
	return sectors, nil
}

func (u *sectorUsecase) GetByID(ctx context.Context, id string) (domain.Sector, error) {
	s, err := u.repo.GetByID(ctx, strings.TrimSpace(id))
	if err != nil {
		return domain.Sector{}, fmt.Errorf("sector get: %w", err)
	}
	return s, nil
}

func (u *sectorUsecase) Create(ctx context.Context, name string) (domain.Sector, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.Sector{}, fmt.Errorf("sector create: %w", domain.ErrInvalidInput)
	}
	s, err := u.repo.Create(ctx, name)
	if err != nil {
		return domain.Sector{}, fmt.Errorf("sector create: %w", err)
	}
	return s, nil
}

func (u *sectorUsecase) Update(ctx context.Context, id, name string) error {
	name = strings.TrimSpace(name)
	if strings.TrimSpace(id) == "" || name == "" {
		return fmt.Errorf("sector update: %w", domain.ErrInvalidInput)
	}
	if err := u.repo.Update(ctx, strings.TrimSpace(id), name); err != nil {
		return fmt.Errorf("sector update: %w", err)
	}
	return nil
}

func (u *sectorUsecase) SoftDelete(ctx context.Context, id string) error {
	if err := u.repo.SoftDelete(ctx, strings.TrimSpace(id)); err != nil {
		return fmt.Errorf("sector soft delete: %w", err)
	}
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd apps/core && go generate ./internal/usecase/... && go test ./internal/usecase/ -run TestSectorUsecase -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/core/internal/usecase/sector.go apps/core/internal/usecase/sector_test.go apps/core/internal/mocks/mock_sector_usecase.go
git commit -m "feat(core/sectors): usecase with trim validation"
```

---

### Task 4: Handler + routes + wiring + tests

**Files:**
- Create: `apps/core/internal/delivery/http/sector/handler.go`
- Create: `apps/core/internal/delivery/http/sector/handler_test.go`
- Modify: `apps/core/internal/delivery/http/router.go`
- Modify: `apps/core/internal/container/container.go`
- Test: `apps/core/internal/delivery/http/sector/handler_test.go`

**Interfaces:**
- Consumes: `usecase.SectorUsecase` (Task 3), `response` envelope, `validator`, `appmw.RequirePermission`
- Produces: 5 wired routes — `GET /api/v1/sectors`, `GET /api/v1/sectors/{id}` (`stocks:read`); `POST /api/v1/admin/sectors`, `PATCH /api/v1/admin/sectors/{id}`, `DELETE /api/v1/admin/sectors/{id}` (`stocks:write`)

- [ ] **Step 1: Write the failing test**

Create `apps/core/internal/delivery/http/sector/handler_test.go`:

```go
package sector

import (
	"bytes"
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
	"github.com/nofendian17/sbterm/libs/pkg/validator"
)

func withID(r *http.Request, id string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func TestSectorHandler_List(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	uc := mocks.NewMockSectorUsecase(ctrl)
	uc.EXPECT().List(gomock.Any()).
		Return([]domain.Sector{{ID: "s1", Name: "Financials"}}, nil)

	handler := NewSectorHandler(uc, validator.New())
	rec := httptest.NewRecorder()
	handler.List(rec, httptest.NewRequest(http.MethodGet, "/api/v1/sectors", nil))

	require.Equal(t, http.StatusOK, rec.Code)
	var env struct {
		Success bool `json:"success"`
		Data    []struct {
			Name string `json:"name"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	require.Len(t, env.Data, 1)
	assert.Equal(t, "Financials", env.Data[0].Name)
}

func TestSectorHandler_Create_Duplicate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	uc := mocks.NewMockSectorUsecase(ctrl)
	uc.EXPECT().Create(gomock.Any(), "Financials").
		Return(domain.Sector{}, domain.ErrSectorNameTaken)

	handler := NewSectorHandler(uc, validator.New())
	body, _ := json.Marshal(createSectorRequest{Name: "Financials"})
	rec := httptest.NewRecorder()
	handler.Create(rec, httptest.NewRequest(http.MethodPost, "/api/v1/admin/sectors", bytes.NewReader(body)))

	assert.Equal(t, http.StatusConflict, rec.Code)
}

func TestSectorHandler_Delete_HasStocks(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	uc := mocks.NewMockSectorUsecase(ctrl)
	uc.EXPECT().SoftDelete(gomock.Any(), "s1").
		Return(domain.ErrSectorHasStocks)

	handler := NewSectorHandler(uc, validator.New())
	req := withID(httptest.NewRequest(http.MethodDelete, "/api/v1/admin/sectors/s1", nil), "s1")
	rec := httptest.NewRecorder()
	handler.Delete(rec, req)

	assert.Equal(t, http.StatusConflict, rec.Code)
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd apps/core && go test ./internal/delivery/http/sector/ -v
```

Expected: FAIL — package `sector` does not exist.

- [ ] **Step 3: Write minimal implementation**

Create `apps/core/internal/delivery/http/sector/handler.go` (mirror `stock/handler.go` style):

```go
// Package sector provides HTTP handlers for the sectors master table.

package sector

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
	"github.com/nofendian17/sbterm/apps/core/internal/usecase"
	"github.com/nofendian17/sbterm/libs/pkg/response"
	"github.com/nofendian17/sbterm/libs/pkg/validator"
)

// SectorHandler serves sectors. Reads are user-facing (stocks:read);
// writes are admin-gated by the router (stocks:write).
type SectorHandler struct {
	uc usecase.SectorUsecase
	v  validator.Validator
}

func NewSectorHandler(uc usecase.SectorUsecase, v validator.Validator) *SectorHandler {
	return &SectorHandler{uc: uc, v: v}
}

type sectorResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (h *SectorHandler) List(w http.ResponseWriter, r *http.Request) {
	sectors, err := h.uc.List(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "internal error")
		return
	}
	resp := make([]sectorResponse, len(sectors))
	for i, s := range sectors {
		resp[i] = sectorResponse{ID: s.ID, Name: s.Name}
	}
	response.OK(w, resp)
}

func (h *SectorHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "id is required")
		return
	}
	s, err := h.uc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrSectorNotFound) {
			response.Error(w, http.StatusNotFound, response.CodeNotFound, "sector not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "internal error")
		return
	}
	response.OK(w, sectorResponse{ID: s.ID, Name: s.Name})
}

type createSectorRequest struct {
	Name string `json:"name" validate:"required"`
}

func (h *SectorHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createSectorRequest
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
	s, err := h.uc.Create(r.Context(), req.Name)
	if err != nil {
		mapSectorError(w, err)
		return
	}
	response.Created(w, sectorResponse{ID: s.ID, Name: s.Name})
}

type updateSectorRequest struct {
	Name string `json:"name" validate:"required"`
}

func (h *SectorHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "id is required")
		return
	}
	var req updateSectorRequest
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
	if err := h.uc.Update(r.Context(), id, req.Name); err != nil {
		mapSectorError(w, err)
		return
	}
	response.Message(w, http.StatusOK, "sector updated")
}

func (h *SectorHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "id is required")
		return
	}
	if err := h.uc.SoftDelete(r.Context(), id); err != nil {
		mapSectorError(w, err)
		return
	}
	response.NoContent(w)
}

func mapSectorError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		response.Error(w, http.StatusUnprocessableEntity, response.CodeValidation, "validation failed")
	case errors.Is(err, domain.ErrSectorNotFound):
		response.Error(w, http.StatusNotFound, response.CodeNotFound, "sector not found")
	case errors.Is(err, domain.ErrSectorNameTaken):
		response.Error(w, http.StatusConflict, response.CodeConflict, "sector name already exists")
	case errors.Is(err, domain.ErrSectorHasStocks):
		response.Error(w, http.StatusConflict, response.CodeConflict, "sector has stocks")
	default:
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "internal error")
	}
}
```

Wire-up (follow the stock pattern exactly):

1. `router.go`: add `Sector *sector.SectorHandler` to `Handlers`; in the `stocks:read` group add `r.Get("/sectors", hs.Sector.List)` and `r.Get("/sectors/{id}", hs.Sector.GetByID)`; in the `stocks:write` group add `r.Post("/sectors", hs.Sector.Create)`, `r.Patch("/sectors/{id}", hs.Sector.Update)`, `r.Delete("/sectors/{id}", hs.Sector.Delete)`.
2. `container.go`: `do.Provide` for `usecase.SectorUsecase` via `NewSectorUsecase(do.MustInvoke[repository.SectorRepository](i))`; `do.Provide` for `*sector.SectorHandler`; add `Sector:` to the `NewRouter` call.

- [ ] **Step 4: Run tests to verify all pass**

```bash
cd apps/core && go build ./... && go test ./internal/delivery/http/sector/ ./internal/usecase/ ./internal/infrastructure/repository/ -v 2>&1 | tail -15
```

Expected: PASS all three packages.

- [ ] **Step 5: Commit**

```bash
git add apps/core/internal/delivery/http/sector/ apps/core/internal/delivery/http/router.go apps/core/internal/container/container.go
git commit -m "feat(core/sectors): HTTP CRUD wired on stocks perms"
```
