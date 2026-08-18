package usecase

import (
	"context"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/repository"
)

//go:generate go run go.uber.org/mock/mockgen -source=fundachart_metrics.go -destination=../mocks/mock_fundachart_metrics_usecase.go -package=mocks -typed
type FundaChartMetricsUsecase interface {
	GetFundaChartMetrics(ctx context.Context, metricName string) ([]domain.FundaChartMetric, error)
}

type fundaChartMetricsUsecase struct {
	repo repository.FundaChartMetricsRepository
}

func NewFundaChartMetricsUsecase(repo repository.FundaChartMetricsRepository) *fundaChartMetricsUsecase {
	return &fundaChartMetricsUsecase{repo: repo}
}

func (u *fundaChartMetricsUsecase) GetFundaChartMetrics(ctx context.Context, metricName string) ([]domain.FundaChartMetric, error) {
	return u.repo.GetFundaChartMetrics(ctx, metricName)
}
