package repository

import (
	"context"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
)

//go:generate go run go.uber.org/mock/mockgen -source=stock.go -destination=../mocks/mock_stock_repository.go -package=mocks -typed

// StockRepository is the storage port for the stock catalog. Every read
// filters soft-deleted rows; driver errors are mapped to domain sentinels
// (ErrStockNotFound, ErrStockSymbolTaken).
type StockRepository interface {
	// GetBySymbol returns the non-deleted stock with the given symbol, or
	// domain.ErrStockNotFound. The returned Stock includes the joined sector.
	GetBySymbol(ctx context.Context, symbol string) (domain.Stock, error)

	// List returns a page of non-deleted stocks matching the filter plus the
	// total count. Page/Limit default to 1/20 and are capped at 100.
	List(ctx context.Context, filter domain.StockFilter) ([]domain.Stock, int, error)

	// Create inserts a new stock. A conflicting primary key (unique
	// violation 23505) maps to domain.ErrStockSymbolTaken.
	Create(ctx context.Context, stock domain.Stock) error

	// Update applies a partial patch. updated_at is set to now(); clearing a
	// nullable column is expressed by an empty-string pointer. Returns
	// domain.ErrStockNotFound when no live row matches.
	Update(ctx context.Context, symbol string, patch domain.StockPatch) error

	// SoftDelete sets deleted_at. The row stays in the table but is
	// invisible to reads. Refused with domain.ErrStockHasWatchlists when
	// live watchlist rows still reference the symbol; returns
	// domain.ErrStockNotFound when missing.
	SoftDelete(ctx context.Context, symbol string) error

	// Upsert inserts a new stock or updates an existing one with the same
	// symbol (used by SyncAll). Only sync-owned fields are overwritten:
	// name, icon_url, is_active, synced_at. exchange and sector_id are
	// admin-managed — the upstream /stocks payload carries neither, so
	// sync never touches them. A soft-deleted row matching an upstream
	// symbol is reactivated (deleted_at = NULL). Returns true when the
	// row was freshly created, false when it already existed (unchanged
	// rows are a no-op and also return false).
	Upsert(ctx context.Context, stock domain.Stock) (created bool, err error)
}
