package repository

import (
	"context"

	"github.com/nofendian17/sbterm-server/internal/domain"
)

//go:generate go run go.uber.org/mock/mockgen -source=mover.go -destination=../mocks/mock_mover_repository.go -package=mocks -typed
type MarketMoverRepository interface {
	GetMarketMover(ctx context.Context, moverType string, filterStocks []string) ([]domain.MarketMover, error)
}
