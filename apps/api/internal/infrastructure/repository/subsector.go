package repository

import (
	"context"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/repository"
	"github.com/nofendian17/sbterm/libs/stockbit"
)

// SubsectorRepository fetches subsector companies from the Stockbit API.
type SubsectorRepository struct {
	client *stockbit.Client
}

func NewSubsectorRepository(client *stockbit.Client) *SubsectorRepository {
	return &SubsectorRepository{client: client}
}

func (r *SubsectorRepository) GetCompanies(ctx context.Context, sectorID, subsectorID string) ([]domain.SubsectorCompany, error) {
	resp, err := r.client.GetSubsectorCompanies(ctx, sectorID, subsectorID)
	if err != nil {
		return nil, err
	}
	companies := make([]domain.SubsectorCompany, 0, len(resp.Data))
	for _, c := range resp.Data {
		companies = append(companies, domain.SubsectorCompany{
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
	return companies, nil
}

var _ repository.SubsectorRepository = (*SubsectorRepository)(nil)
