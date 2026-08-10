package repository

import (
	"context"

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/infrastructure/stockbit"
	"github.com/nofendian17/sbterm-server/internal/repository"
)

// RunningTradeRepository fetches running trade charts from the Stockbit API.
type RunningTradeRepository struct {
	client *stockbit.Client
}

func NewRunningTradeRepository(client *stockbit.Client) *RunningTradeRepository {
	return &RunningTradeRepository{client: client}
}

func (r *RunningTradeRepository) GetRunningTradeChart(ctx context.Context, symbol string, brokerCodes []string, from, to, investorType, marketBoard, period string) (*domain.RunningTradeData, error) {
	resp, err := r.client.GetRunningTradeChart(ctx, symbol, brokerCodes, from, to, investorType, marketBoard, period)
	if err != nil {
		return nil, err
	}
	d := resp.Data
	out := &domain.RunningTradeData{
		From:            d.From,
		To:              d.To,
		DataLastUpdated: d.DataLastUpdated,
		PriceChartData:  make([]domain.RunningTradePricePoint, 0, len(d.PriceChartData)),
		BrokerChartData: make([]domain.RunningTradeBrokerGroup, 0, len(d.BrokerChartData)),
		DateSessionInfo: d.DateSessionInfo,
	}
	for _, p := range d.PriceChartData {
		out.PriceChartData = append(out.PriceChartData, domain.RunningTradePricePoint{
			Date:          p.Date,
			Time:          p.Time,
			Value:         toRawFormatted(p.Value),
			DatetimeLabel: p.DatetimeLabel,
			Open:          toRawFormattedPtr(p.Open),
			High:          toRawFormattedPtr(p.High),
			Low:           toRawFormattedPtr(p.Low),
		})
	}
	for _, g := range d.BrokerChartData {
		group := domain.RunningTradeBrokerGroup{
			Type:    g.Type,
			Brokers: g.Brokers,
			Charts:  make([]domain.RunningTradeBrokerChart, 0, len(g.Charts)),
		}
		for _, ch := range g.Charts {
			group.Charts = append(group.Charts, domain.RunningTradeBrokerChart{
				BrokerCode: ch.BrokerCode,
				Chart:      toRunningTradeChartPoints(ch.Chart),
			})
		}
		out.BrokerChartData = append(out.BrokerChartData, group)
	}
	return out, nil
}

func toRunningTradeChartPoints(points []stockbit.RunningTradeChartPoint) []domain.RunningTradeChartPoint {
	out := make([]domain.RunningTradeChartPoint, 0, len(points))
	for _, p := range points {
		out = append(out, domain.RunningTradeChartPoint{
			Date:          p.Date,
			Time:          p.Time,
			Value:         toRawFormatted(p.Value),
			DatetimeLabel: p.DatetimeLabel,
			Open:          toRawFormattedPtr(p.Open),
			High:          toRawFormattedPtr(p.High),
			Low:           toRawFormattedPtr(p.Low),
		})
	}
	return out
}

func toRawFormatted(v stockbit.RunningTradeValue) domain.RawFormatted {
	return domain.RawFormatted{Raw: v.Raw, Formatted: v.Formatted}
}

func toRawFormattedPtr(v *stockbit.RunningTradeValue) *domain.RawFormatted {
	if v == nil {
		return nil
	}
	rf := toRawFormatted(*v)
	return &rf
}

var _ repository.RunningTradeRepository = (*RunningTradeRepository)(nil)
