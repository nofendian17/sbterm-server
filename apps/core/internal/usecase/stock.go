// Package usecase implements the business logic for the core domain.

package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
	"github.com/nofendian17/sbterm/apps/core/internal/repository"
	"github.com/nofendian17/sbterm/libs/pkg/log"
)

//go:generate go run go.uber.org/mock/mockgen -source=stock.go -destination=../mocks/mock_stock_usecase.go -package=mocks -typed

// StockUsecase manages the stock catalog. Users call List and GetBySymbol;
// admins call Create, Update, SoftDelete and SyncAll.
type StockUsecase interface {
	List(ctx context.Context, filter domain.StockFilter) ([]domain.Stock, int, error)
	GetBySymbol(ctx context.Context, symbol string) (domain.Stock, error)
	Create(ctx context.Context, input domain.StockCreateInput) (domain.Stock, error)
	Update(ctx context.Context, symbol string, input domain.StockUpdateInput) error
	SoftDelete(ctx context.Context, symbol string) error
	SyncAll(ctx context.Context) (domain.StockSyncResult, error)
}

type stockUsecase struct {
	repo    repository.StockRepository
	sectors repository.SectorRepository
	sync    repository.StockSyncClient
	log     log.Logger
}

// NewStockUsecase wires up the stock usecase.
func NewStockUsecase(
	repo repository.StockRepository,
	sectors repository.SectorRepository,
	sync repository.StockSyncClient,
	logger log.Logger,
) StockUsecase {
	return &stockUsecase{repo: repo, sectors: sectors, sync: sync, log: logger}
}

func (u *stockUsecase) List(ctx context.Context, filter domain.StockFilter) ([]domain.Stock, int, error) {
	stocks, total, err := u.repo.List(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("stock list: %w", err)
	}
	return stocks, total, nil
}

func (u *stockUsecase) GetBySymbol(ctx context.Context, symbol string) (domain.Stock, error) {
	s, err := u.repo.GetBySymbol(ctx, strings.ToUpper(symbol))
	if err != nil {
		return domain.Stock{}, fmt.Errorf("stock get: %w", err)
	}
	return s, nil
}

func (u *stockUsecase) Create(ctx context.Context, input domain.StockCreateInput) (domain.Stock, error) {
	input.Symbol = strings.TrimSpace(input.Symbol)
	input.Name = strings.TrimSpace(input.Name)
	if input.Symbol == "" || input.Name == "" {
		return domain.Stock{}, fmt.Errorf("stock create: %w", domain.ErrInvalidInput)
	}

	sectorID, err := u.resolveSector(ctx, input.Sector)
	if err != nil {
		return domain.Stock{}, fmt.Errorf("stock create: %w", err)
	}

	isActive := true
	if input.IsActive != nil {
		isActive = *input.IsActive
	}
	s := domain.Stock{
		Symbol:   strings.ToUpper(input.Symbol),
		Name:     input.Name,
		SectorID: sectorID,
		Exchange: input.Exchange,
		IconURL:  input.IconURL,
		IsActive: isActive,
	}
	if err := u.repo.Create(ctx, s); err != nil {
		return domain.Stock{}, fmt.Errorf("stock create: %w", err)
	}
	// Re-read the persisted row so the response carries DB-generated values
	// (created_at/updated_at, sector join).
	created, err := u.repo.GetBySymbol(ctx, s.Symbol)
	if err != nil {
		return domain.Stock{}, fmt.Errorf("stock create: reload: %w", err)
	}
	return created, nil
}

func (u *stockUsecase) Update(ctx context.Context, symbol string, input domain.StockUpdateInput) error {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if symbol == "" {
		return fmt.Errorf("stock update: %w", domain.ErrInvalidInput)
	}

	patch := domain.StockPatch{
		Name:     input.Name,
		Exchange: input.Exchange,
		IconURL:  input.IconURL,
		IsActive: input.IsActive,
	}
	if patch.Name != nil {
		trimmed := strings.TrimSpace(*patch.Name)
		if trimmed == "" {
			return fmt.Errorf("stock update: %w", domain.ErrInvalidInput)
		}
		patch.Name = &trimmed
	}
	if input.Sector != nil {
		patch.SectorSet = true
		if strings.TrimSpace(*input.Sector) != "" {
			sector, err := u.sectors.GetByName(ctx, strings.TrimSpace(*input.Sector))
			if err != nil {
				return fmt.Errorf("stock update: %w", err)
			}
			id := sector.ID
			patch.SectorID = &id
		}
	}

	if patch.Name == nil && !patch.SectorSet && patch.Exchange == nil &&
		patch.IconURL == nil && patch.IsActive == nil {
		return nil // no-op
	}
	if err := u.repo.Update(ctx, symbol, patch); err != nil {
		return fmt.Errorf("stock update: %w", err)
	}
	return nil
}

func (u *stockUsecase) SoftDelete(ctx context.Context, symbol string) error {
	if err := u.repo.SoftDelete(ctx, strings.ToUpper(strings.TrimSpace(symbol))); err != nil {
		return fmt.Errorf("stock soft delete: %w", err)
	}
	return nil
}

// SyncAll refreshes the catalog from the configured upstream. It is
// best-effort per symbol: an error on one stock is recorded in
// result.Errors, never returned. The function only errors when the upstream
// call itself failed or the sync client is not configured.
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
			res.Updated++
		}
	}
	if res.Failed > 0 {
		u.log.Warn("stock sync completed with errors",
			"fetched", res.Fetched, "created", res.Created,
			"updated", res.Updated, "failed", res.Failed)
	}
	return res, nil
}

// resolveSector maps an optional sector *name* to a sector id, or nil when
// the input is nil/empty (stock has no sector). Unknown names error with
// domain.ErrSectorNotFound.
func (u *stockUsecase) resolveSector(ctx context.Context, sectorName *string) (*string, error) {
	if sectorName == nil || strings.TrimSpace(*sectorName) == "" {
		return nil, nil
	}
	sector, err := u.sectors.GetByName(ctx, strings.TrimSpace(*sectorName))
	if err != nil {
		if errors.Is(err, domain.ErrSectorNotFound) {
			return nil, err
		}
		return nil, err
	}
	id := sector.ID
	return &id, nil
}
