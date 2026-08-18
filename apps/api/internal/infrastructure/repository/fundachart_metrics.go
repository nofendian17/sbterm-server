package repository

import (
	"context"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/repository"
	"github.com/nofendian17/sbterm/libs/stockbit"
)

// FundaChartMetricsRepository fetches the metrics catalog from the Stockbit
// API.
type FundaChartMetricsRepository struct {
	client *stockbit.Client
}

func NewFundaChartMetricsRepository(client *stockbit.Client) *FundaChartMetricsRepository {
	return &FundaChartMetricsRepository{client: client}
}

func (r *FundaChartMetricsRepository) GetFundaChartMetrics(ctx context.Context, metricName string) ([]domain.FundaChartMetric, error) {
	resp, err := r.client.GetFundaChartMetrics(ctx, metricName)
	if err != nil {
		return nil, err
	}
	out := make([]domain.FundaChartMetric, 0, len(resp.Data))
	for _, m := range resp.Data {
		out = append(out, toMetricDomain(m))
	}
	return out, nil
}

func toMetricDomain(m stockbit.FundaChartMetric) domain.FundaChartMetric {
	dm := domain.FundaChartMetric{
		FitemID:       m.FitemID,
		FitemName:     m.FitemName,
		ShowChartIcon: m.ShowChartIcon,
		Child:         make([]domain.FundaChartMetric, 0, len(m.Child)),
	}
	for _, c := range m.Child {
		dm.Child = append(dm.Child, toMetricDomain(c))
	}
	return dm
}

var _ repository.FundaChartMetricsRepository = (*FundaChartMetricsRepository)(nil)
