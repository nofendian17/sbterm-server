package repository

import (
	"context"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/repository"
	"github.com/nofendian17/sbterm/libs/stockbit"
)

// FundaChartRepository fetches the raw fundamental chart series from the
// Stockbit API.
type FundaChartRepository struct {
	client *stockbit.Client
}

func NewFundaChartRepository(client *stockbit.Client) *FundaChartRepository {
	return &FundaChartRepository{client: client}
}

func (r *FundaChartRepository) GetFundaChart(ctx context.Context, symbol, item, timeframe string) ([]domain.FundaChartCompany, error) {
	resp, err := r.client.GetFundaChart(ctx, symbol, item, timeframe)
	if err != nil {
		return nil, err
	}
	out := make([]domain.FundaChartCompany, 0, len(resp.Data))
	for _, c := range resp.Data {
		company := domain.FundaChartCompany{CompanyID: c.CompanyID, CompanyName: c.CompanyName, Ratios: make([]domain.FundaChartRatio, 0, len(c.Ratios))}
		for _, rt := range c.Ratios {
			ratio := domain.FundaChartRatio{
				DecimalPoint: rt.DecimalPoint,
				GroupData:    rt.GroupData,
				ItemID:       rt.ItemID,
				ItemName:     rt.ItemName,
				ItemType:     rt.ItemType,
				Suffix:       rt.Suffix,
				XAxisID:      rt.XAxisID,
				YAxisID:      rt.YAxisID,
				ChartData:    make([]domain.FundaChartPoint, 0, len(rt.ChartData)),
			}
			for _, p := range rt.ChartData {
				ratio.ChartData = append(ratio.ChartData, domain.FundaChartPoint{
					Date:         p.Date,
					FormatedDate: p.FormatedDate,
					Value:        p.Value,
					RatioValue:   p.RatioValue,
				})
			}
			company.Ratios = append(company.Ratios, ratio)
		}
		out = append(out, company)
	}
	return out, nil
}

var _ repository.FundaChartRepository = (*FundaChartRepository)(nil)
