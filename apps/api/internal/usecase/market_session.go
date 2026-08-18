package usecase

import (
	"context"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/repository"
)

//go:generate go run go.uber.org/mock/mockgen -source=market_session.go -destination=../mocks/mock_market_session_usecase.go -package=mocks -typed
type MarketSessionUsecase interface {
	GetMarketSession(ctx context.Context) (*domain.MarketSession, error)
}

type marketSessionUsecase struct {
	repo repository.MarketSessionRepository
}

func NewMarketSessionUsecase(repo repository.MarketSessionRepository) *marketSessionUsecase {
	return &marketSessionUsecase{repo: repo}
}

func (u *marketSessionUsecase) GetMarketSession(ctx context.Context) (*domain.MarketSession, error) {
	return u.repo.GetMarketSession(ctx)
}
