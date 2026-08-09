package usecase

import (
	"context"

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/repository"
)

//go:generate go run go.uber.org/mock/mockgen -source=fundachart.go -destination=../mocks/mock_fundachart_usecase.go -package=mocks -typed
type FundaChartUsecase interface {
	GetFundaChart(ctx context.Context, symbol, item, timeframe string) ([]domain.FundaChartCompany, error)
}

type fundaChartUsecase struct {
	repo repository.FundaChartRepository
}

func NewFundaChartUsecase(repo repository.FundaChartRepository) *fundaChartUsecase {
	return &fundaChartUsecase{repo: repo}
}

func (u *fundaChartUsecase) GetFundaChart(ctx context.Context, symbol, item, timeframe string) ([]domain.FundaChartCompany, error) {
	return u.repo.GetFundaChart(ctx, symbol, item, timeframe)
}
