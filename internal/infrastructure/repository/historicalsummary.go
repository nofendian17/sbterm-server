package repository

import (
	"context"

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/infrastructure/stockbit"
	"github.com/nofendian17/sbterm-server/internal/repository"
)

// HistoricalSummaryRepository fetches historical price summaries from the Stockbit API.
type HistoricalSummaryRepository struct {
	client *stockbit.Client
}

func NewHistoricalSummaryRepository(client *stockbit.Client) *HistoricalSummaryRepository {
	return &HistoricalSummaryRepository{client: client}
}

func (r *HistoricalSummaryRepository) GetHistoricalSummary(ctx context.Context, symbol, period, startDate, endDate string, limit, page int) (*domain.HistoricalSummaryData, error) {
	resp, err := r.client.GetHistoricalSummary(ctx, symbol, period, startDate, endDate, limit, page)
	if err != nil {
		return nil, err
	}
	d := resp.Data
	out := &domain.HistoricalSummaryData{
		Result: make([]domain.HistoricalSummaryItem, 0, len(d.Result)),
		Paginate: domain.HistoricalSummaryPaginate{
			NextPage: d.Paginate.NextPage,
		},
	}
	for _, it := range d.Result {
		out.Result = append(out.Result, domain.HistoricalSummaryItem{
			Date:             it.Date,
			Close:            it.Close,
			Change:           it.Change,
			Value:            it.Value,
			Volume:           it.Volume,
			Frequency:        it.Frequency,
			ForeignBuy:       it.ForeignBuy,
			ForeignSell:      it.ForeignSell,
			NetForeign:       it.NetForeign,
			Open:             it.Open,
			High:             it.High,
			Low:              it.Low,
			Average:          it.Average,
			ChangePercentage: it.ChangePercentage,
		})
	}
	return out, nil
}

var _ repository.HistoricalSummaryRepository = (*HistoricalSummaryRepository)(nil)
