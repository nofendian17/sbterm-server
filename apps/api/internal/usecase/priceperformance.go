package usecase

import (
	"context"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/repository"
)

//go:generate go run go.uber.org/mock/mockgen -source=priceperformance.go -destination=../mocks/mock_priceperformance_usecase.go -package=mocks -typed
type PricePerformanceUsecase interface {
	GetPricePerformance(ctx context.Context, symbol string) (*domain.PricePerformanceData, error)
}

type pricePerformanceUsecase struct {
	repo repository.PricePerformanceRepository
}

func NewPricePerformanceUsecase(repo repository.PricePerformanceRepository) *pricePerformanceUsecase {
	return &pricePerformanceUsecase{repo: repo}
}

func (u *pricePerformanceUsecase) GetPricePerformance(ctx context.Context, symbol string) (*domain.PricePerformanceData, error) {
	return u.repo.GetPricePerformance(ctx, symbol)
}
