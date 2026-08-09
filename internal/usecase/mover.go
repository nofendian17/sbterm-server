package usecase

import (
	"context"

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/repository"
)

//go:generate go run go.uber.org/mock/mockgen -source=mover.go -destination=../mocks/mock_mover_usecase.go -package=mocks -typed
type MarketMoverUsecase interface {
	GetMarketMover(ctx context.Context, moverType string, filterStocks []string) ([]domain.MarketMover, error)
}

type marketMoverUsecase struct {
	repo repository.MarketMoverRepository
}

func NewMarketMoverUsecase(repo repository.MarketMoverRepository) *marketMoverUsecase {
	return &marketMoverUsecase{repo: repo}
}

func (u *marketMoverUsecase) GetMarketMover(ctx context.Context, moverType string, filterStocks []string) ([]domain.MarketMover, error) {
	return u.repo.GetMarketMover(ctx, moverType, filterStocks)
}
