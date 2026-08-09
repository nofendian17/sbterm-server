package repository

import (
	"context"

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/infrastructure/stockbit"
	"github.com/nofendian17/sbterm-server/internal/repository"
)

// SubsidiaryRepository fetches the subsidiary list from the Stockbit API.
type SubsidiaryRepository struct {
	client *stockbit.Client
}

func NewSubsidiaryRepository(client *stockbit.Client) *SubsidiaryRepository {
	return &SubsidiaryRepository{client: client}
}

func (r *SubsidiaryRepository) GetSubsidiaries(ctx context.Context, symbol string) (*domain.SubsidiaryData, error) {
	resp, err := r.client.GetSubsidiaries(ctx, symbol)
	if err != nil {
		return nil, err
	}
	out := &domain.SubsidiaryData{
		Currency:          resp.Data.Currency,
		LastUpdatedPeriod: resp.Data.LastUpdatedPeriod,
		Unit:              resp.Data.Unit,
		Subsidiaries:      make([]domain.Subsidiary, 0, len(resp.Data.Subsidiaries)),
	}
	for _, s := range resp.Data.Subsidiaries {
		out.Subsidiaries = append(out.Subsidiaries, domain.Subsidiary{
			CompanyName:       s.CompanyName,
			BusinessType:      s.BusinessType,
			Location:          s.Location,
			CommercialYear:    s.CommercialYear,
			TotalAssets:       s.TotalAssets,
			Percentage:        s.Percentage,
			ID:                s.ID,
			OperationalStatus: s.OperationalStatus,
			Period:            s.Period,
			Raw:               s.Raw,
		})
	}
	return out, nil
}

var _ repository.SubsidiaryRepository = (*SubsidiaryRepository)(nil)
