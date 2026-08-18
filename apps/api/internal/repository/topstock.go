package repository

import (
	"context"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
)

//go:generate go run go.uber.org/mock/mockgen -source=topstock.go -destination=../mocks/mock_topstock_repository.go -package=mocks -typed
type TopStockRepository interface {
	GetTopStock(ctx context.Context, start, end, investorType, marketType, valueType string, page int) (*domain.TopStockData, error)
}
