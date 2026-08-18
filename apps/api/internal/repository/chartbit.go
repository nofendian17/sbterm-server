package repository

import (
	"context"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
)

//go:generate go run go.uber.org/mock/mockgen -source=chartbit.go -destination=../mocks/mock_chartbit_repository.go -package=mocks -typed
type ChartbitRepository interface {
	GetChartPrice(ctx context.Context, symbol, timeframe, from, to string, limit int) (*domain.ChartPriceData, error)
}
