package repository

import (
	"context"

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/infrastructure/stockbit"
	"github.com/nofendian17/sbterm-server/internal/repository"
)

// ihsgSectorID and ihsgSubsectorID identify the IHSG constituents list on the
// upstream API.
//
// ponytail: hardcoded upstream ids; if the API ever exposes them in a
// configuration endpoint, resolve them there instead.
const (
	ihsgSectorID    = "88"
	ihsgSubsectorID = "467"
)

// StocksRepository fetches the IHSG stock list from the Stockbit API.
type StocksRepository struct {
	client *stockbit.Client
}

func NewStocksRepository(client *stockbit.Client) *StocksRepository {
	return &StocksRepository{client: client}
}

func (r *StocksRepository) GetStocks(ctx context.Context) ([]domain.Stock, error) {
	resp, err := r.client.GetSubsectorCompanies(ctx, ihsgSectorID, ihsgSubsectorID)
	if err != nil {
		return nil, err
	}
	stocks := make([]domain.Stock, 0, len(resp.Data))
	for _, c := range resp.Data {
		stocks = append(stocks, domain.Stock{
			Symbol:        c.Symbol,
			Name:          c.Name,
			Last:          c.Last,
			Change:        c.Change,
			Percent:       c.Percent,
			Volume:        c.Volume,
			Value:         c.Value,
			MarketCap:     c.MarketCap,
			IconURL:       c.IconURL,
			CompanyStatus: c.CompanyStatus,
			IsUMA:         c.UMA,
		})
	}
	return stocks, nil
}

var _ repository.StocksRepository = (*StocksRepository)(nil)
