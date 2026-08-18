package repository

import (
	"context"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/repository"
	"github.com/nofendian17/sbterm/libs/stockbit"
)

// TrendingRepository fetches trending stocks from the Stockbit API.
type TrendingRepository struct {
	client *stockbit.Client
}

func NewTrendingRepository(client *stockbit.Client) *TrendingRepository {
	return &TrendingRepository{client: client}
}

func (r *TrendingRepository) GetTrending(ctx context.Context) ([]domain.TrendingStock, error) {
	resp, err := r.client.GetTrending(ctx)
	if err != nil {
		return nil, err
	}
	stocks := make([]domain.TrendingStock, 0, len(resp.Data))
	for _, s := range resp.Data {
		stocks = append(stocks, domain.TrendingStock{
			Symbol:   s.Symbol,
			Name:     s.Name,
			Last:     s.Last,
			Change:   s.Change,
			Percent:  s.Percent,
			Previous: s.Previous,
			LogoURL:  s.IconURL,
			Status:   s.Status,
		})
	}
	return stocks, nil
}

var _ repository.TrendingRepository = (*TrendingRepository)(nil)
