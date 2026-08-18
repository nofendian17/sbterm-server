package repository

import (
	"context"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/repository"
	"github.com/nofendian17/sbterm/libs/stockbit"
)

// IndexRepository fetches market indexes from the Stockbit API.
type IndexRepository struct {
	client *stockbit.Client
}

func NewIndexRepository(client *stockbit.Client) *IndexRepository {
	return &IndexRepository{client: client}
}

func (r *IndexRepository) GetIndexes(ctx context.Context) (*domain.Indexes, error) {
	resp, err := r.client.GetIndexes(ctx)
	if err != nil {
		return nil, err
	}
	return &domain.Indexes{
		Main: toIndexes(resp.Data.Main),
		All:  toIndexes(resp.Data.All),
	}, nil
}

func toIndexes(list []stockbit.Index) []domain.Index {
	out := make([]domain.Index, 0, len(list))
	for _, i := range list {
		out = append(out, domain.Index{
			Symbol:    i.Symbol,
			Name:      i.Name,
			Last:      i.Last,
			Change:    i.Change,
			Percent:   i.Percent,
			MarketCap: i.MarketCap,
		})
	}
	return out
}

var _ repository.IndexRepository = (*IndexRepository)(nil)
