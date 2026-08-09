package repository

import (
	"context"

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/infrastructure/stockbit"
	"github.com/nofendian17/sbterm-server/internal/repository"
)

// ShareholdingCompositionRepository fetches the shareholding composition from
// the Stockbit API.
type ShareholdingCompositionRepository struct {
	client *stockbit.Client
}

func NewShareholdingCompositionRepository(client *stockbit.Client) *ShareholdingCompositionRepository {
	return &ShareholdingCompositionRepository{client: client}
}

func (r *ShareholdingCompositionRepository) GetShareholdingComposition(ctx context.Context, symbol, periodStart, periodEnd string) ([]domain.ShareholdingCompositionPeriod, error) {
	resp, err := r.client.GetShareholdingComposition(ctx, symbol, periodStart, periodEnd)
	if err != nil {
		return nil, err
	}
	out := make([]domain.ShareholdingCompositionPeriod, 0, len(resp.Data.Periods))
	for _, p := range resp.Data.Periods {
		comps := make([]domain.ShareholdingComposition, 0, len(p.Compositions))
		for _, c := range p.Compositions {
			comps = append(comps, domain.ShareholdingComposition{
				Label:      c.Label,
				Shares:     domain.ShareholdingRawFormatted{Raw: c.Shares.Raw, Formatted: c.Shares.Formatted},
				Percentage: domain.ShareholdingPercent{Raw: c.Percentage.Raw, Formatted: c.Percentage.Formatted},
				Colors:     domain.ShareholdingColors{Light: c.Colors.Light, Dark: c.Colors.Dark},
			})
		}
		out = append(out, domain.ShareholdingCompositionPeriod{
			ReportDate:   p.ReportDate,
			TotalShares:  domain.ShareholdingRawFormatted{Raw: p.TotalShares.Raw, Formatted: p.TotalShares.Formatted},
			Compositions: comps,
		})
	}
	return out, nil
}

var _ repository.ShareholdingCompositionRepository = (*ShareholdingCompositionRepository)(nil)
