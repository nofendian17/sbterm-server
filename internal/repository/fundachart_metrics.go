package repository

import (
	"context"

	"github.com/nofendian17/sbterm-server/internal/domain"
)

//go:generate go run go.uber.org/mock/mockgen -source=fundachart_metrics.go -destination=../mocks/mock_fundachart_metrics_repository.go -package=mocks -typed
type FundaChartMetricsRepository interface {
	GetFundaChartMetrics(ctx context.Context, metricName string) ([]domain.FundaChartMetric, error)
}
