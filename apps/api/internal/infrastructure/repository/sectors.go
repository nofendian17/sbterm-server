package repository

import (
	"context"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/repository"
	"github.com/nofendian17/sbterm/libs/stockbit"
)

// SectorsRepository fetches the sector list from the Stockbit API.
type SectorsRepository struct {
	client *stockbit.Client
}

func NewSectorsRepository(client *stockbit.Client) *SectorsRepository {
	return &SectorsRepository{client: client}
}

func (r *SectorsRepository) GetSectors(ctx context.Context) ([]domain.Sector, error) {
	resp, err := r.client.GetSectors(ctx, stockbit.SectorsRequest{})
	if err != nil {
		return nil, err
	}
	sectors := make([]domain.Sector, 0, len(resp.Data.PChangeInfo))
	for _, c := range resp.Data.PChangeInfo {
		sectors = append(sectors, domain.Sector{
			Symbol:  c.Symbol,
			ID:      c.ID,
			Icon:    c.Icon,
			Type:    c.Type,
			Last:    c.Last,
			Change:  c.Change,
			Percent: c.Percent,
		})
	}
	return sectors, nil
}

var _ repository.SectorsRepository = (*SectorsRepository)(nil)
