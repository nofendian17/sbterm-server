package repository

import (
	"context"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
)

//go:generate go run go.uber.org/mock/mockgen -source=stocks.go -destination=../mocks/mock_stocks_repository.go -package=mocks -typed
type StocksRepository interface {
	GetStocks(ctx context.Context) ([]domain.Stock, error)
}
