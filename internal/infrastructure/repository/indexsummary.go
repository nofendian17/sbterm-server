package repository

import (
	"context"

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/infrastructure/stockbit"
	"github.com/nofendian17/sbterm-server/internal/repository"
)

// IndexSummaryRepository fetches the index chart summary from the Stockbit API.
type IndexSummaryRepository struct {
	client *stockbit.Client
}

func NewIndexSummaryRepository(client *stockbit.Client) *IndexSummaryRepository {
	return &IndexSummaryRepository{client: client}
}

func (r *IndexSummaryRepository) GetIndexSummary(ctx context.Context, symbol, from, to, interval string) (*domain.IndexSummaryData, error) {
	resp, err := r.client.GetIndexSummary(ctx, symbol, from, to, interval)
	if err != nil {
		return nil, err
	}
	out := &domain.IndexSummaryData{
		Cagr:                   resp.Data.Cagr,
		Change:                 float64(resp.Data.Change),
		Drawdown:               resp.Data.Drawdown,
		MarkingPoint:           resp.Data.MarkingPoint,
		Percentage:             resp.Data.Percentage,
		Timeframe:              resp.Data.Timeframe,
		XAxisOpt:               resp.Data.XAxisOpt,
		Previous:               float64(resp.Data.Previous),
		LineWeight:             float64(resp.Data.LineWeight),
		PreviousTimeframePrice: mapIndexSummaryPrice(resp.Data.PreviousTimeframePrice),
		ChartType:              resp.Data.ChartType,
		IntervalInMinutes:      resp.Data.IntervalInMinutes,
		AllowedChartType:       resp.Data.AllowedChartType,
		MaxCandles:             resp.Data.MaxCandles,
		Prices:                 make([]domain.IndexSummaryPrice, 0, len(resp.Data.Prices)),
	}
	for _, p := range resp.Data.Prices {
		out.Prices = append(out.Prices, mapIndexSummaryPrice(p))
	}
	return out, nil
}

func mapIndexSummaryPrice(p stockbit.IndexSummaryPrice) domain.IndexSummaryPrice {
	return domain.IndexSummaryPrice{
		Date:          p.Date,
		FormattedDate: p.FormattedDate,
		XLabel:        p.XLabel,
		Value:         p.Value,
		Percentage:    p.Percentage,
		Change:        float64(p.Change),
		Open:          p.Open,
		High:          p.High,
		Low:           p.Low,
		Volume:        p.Volume,
	}
}

var _ repository.IndexSummaryRepository = (*IndexSummaryRepository)(nil)
