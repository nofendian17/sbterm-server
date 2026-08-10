package usecase

import (
	"context"

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/repository"
)

//go:generate go run go.uber.org/mock/mockgen -source=chartbit.go -destination=../mocks/mock_chartbit_usecase.go -package=mocks -typed
type ChartbitUsecase interface {
	GetChartPrice(ctx context.Context, symbol, timeframe, from, to string, limit int) (*domain.ChartPriceData, error)
}

type chartbitUsecase struct {
	repo repository.ChartbitRepository
}

func NewChartbitUsecase(repo repository.ChartbitRepository) *chartbitUsecase {
	return &chartbitUsecase{repo: repo}
}

func (u *chartbitUsecase) GetChartPrice(ctx context.Context, symbol, timeframe, from, to string, limit int) (*domain.ChartPriceData, error) {
	return u.repo.GetChartPrice(ctx, symbol, timeframe, from, to, limit)
}
