package repository

import (
	"context"

	"github.com/nofendian17/sbterm/apps/core/internal/domain"
)

//go:generate go run go.uber.org/mock/mockgen -source=stock_sync_client.go -destination=../mocks/mock_stock_sync_client.go -package=mocks -typed

// StockSyncClient is the upstream-data port the StockUsecase depends on to
// refresh the local catalog. The concrete implementation calls the apps/api
// endpoint documented in docs/api.md (GET /api/v1/stocks).
type StockSyncClient interface {
	// ListSymbols fetches the current catalog from upstream. Implementations
	// apply a request-scoped timeout and must NOT swallow upstream errors.
	ListSymbols(ctx context.Context) ([]domain.Stock, error)
}
