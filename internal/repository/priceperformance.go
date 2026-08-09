package repository

import (
	"context"

	"github.com/nofendian17/sbterm-server/internal/domain"
)

//go:generate go run go.uber.org/mock/mockgen -source=priceperformance.go -destination=../mocks/mock_priceperformance_repository.go -package=mocks -typed
type PricePerformanceRepository interface {
	GetPricePerformance(ctx context.Context, symbol string) (*domain.PricePerformanceData, error)
}
