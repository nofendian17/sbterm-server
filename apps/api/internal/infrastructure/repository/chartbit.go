package repository

import (
	"context"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/repository"
	"github.com/nofendian17/sbterm/libs/stockbit"
)

// ChartbitRepository fetches chart price bars from the Stockbit API.
type ChartbitRepository struct {
	client *stockbit.Client
}

func NewChartbitRepository(client *stockbit.Client) *ChartbitRepository {
	return &ChartbitRepository{client: client}
}

func (r *ChartbitRepository) GetChartPrice(ctx context.Context, symbol, timeframe, from, to string, limit int) (*domain.ChartPriceData, error) {
	resp, err := r.client.GetChartPrice(ctx, symbol, timeframe, from, to, limit)
	if err != nil {
		return nil, err
	}
	out := &domain.ChartPriceData{
		AllowDecimal: resp.Data.AllowDecimal,
		Chartbit:     make([]domain.ChartPrice, 0, len(resp.Data.Chartbit)),
	}
	for _, p := range resp.Data.Chartbit {
		out.Chartbit = append(out.Chartbit, domain.ChartPrice{
			Date:             p.Date,
			Unixdate:         p.Unixdate,
			Datetime:         p.Datetime,
			UnixTimestamp:    p.UnixTimestamp,
			Open:             p.Open,
			High:             p.High,
			Low:              p.Low,
			Close:            p.Close,
			Volume:           float64(p.Volume),
			Value:            p.Value,
			Frequency:        float64(p.Frequency),
			ForeignBuy:       p.ForeignBuy,
			ForeignSell:      p.ForeignSell,
			ForeignFlow:      p.ForeignFlow,
			SoxClose:         p.SoxClose,
			Dividend:         p.Dividend,
			ShareOutstanding: p.ShareOutstanding,
			FreqAnalyzer:     p.FreqAnalyzer,
			Lot:              p.Lot,
			ForeignBuyToday:  p.ForeignBuyToday,
			ForeignSellToday: p.ForeignSellToday,
			Symbol:           p.Symbol,
		})
	}
	return out, nil
}

var _ repository.ChartbitRepository = (*ChartbitRepository)(nil)
