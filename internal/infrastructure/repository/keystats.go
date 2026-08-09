package repository

import (
	"context"

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/infrastructure/stockbit"
	"github.com/nofendian17/sbterm-server/internal/repository"
)

// KeystatsRepository fetches key-stats ratios from the Stockbit API.
type KeystatsRepository struct {
	client *stockbit.Client
}

func NewKeystatsRepository(client *stockbit.Client) *KeystatsRepository {
	return &KeystatsRepository{client: client}
}

func (r *KeystatsRepository) GetKeystats(ctx context.Context, symbol string, yearLimit int) (*domain.Keystats, error) {
	resp, err := r.client.GetKeystats(ctx, symbol, yearLimit)
	if err != nil {
		return nil, err
	}
	return toKeystatsDomain(resp.Data), nil
}

func toKeystatsDomain(d stockbit.KeystatsData) *domain.Keystats {
	out := &domain.Keystats{
		Info:                    d.Info,
		Stats:                   toKeystatsStatsDomain(d.Stats),
		FinancialYearParent:     toYearParentDomain(d.FinancialYearParent),
		DividendGroup:           toDividendGroupDomain(d.DividendGroup),
		FinancialReportCurrency: d.FinancialReportCurrency,
		ClosureFinItemsResults:  make([]domain.KeystatsFinGroup, 0, len(d.ClosureFinItemsResults)),
	}
	for _, g := range d.ClosureFinItemsResults {
		group := domain.KeystatsFinGroup{
			KeystatsName:   g.KeystatsName,
			FinNameResults: make([]domain.KeystatsItem, 0, len(g.FinNameResults)),
		}
		for _, it := range g.FinNameResults {
			group.FinNameResults = append(group.FinNameResults, domain.KeystatsItem{
				Fitem:          domain.KeystatsFitem{ID: it.Fitem.ID, Name: it.Fitem.Name, Value: it.Fitem.Value},
				IsNewUpdate:    it.IsNewUpdate,
				HiddenGraphIco: it.HiddenGraphIco,
			})
		}
		out.ClosureFinItemsResults = append(out.ClosureFinItemsResults, group)
	}
	return out
}

func toKeystatsStatsDomain(s stockbit.KeystatsStats) domain.KeystatsStats {
	return domain.KeystatsStats{
		CurrentShareOutstanding: s.CurrentShareOutstanding,
		MarketCap:               s.MarketCap,
		EnterpriseValue:         s.EnterpriseValue,
		FreeFloat:               s.FreeFloat,
	}
}

func toYearParentDomain(p stockbit.KeystatsYearParent) domain.KeystatsYearParent {
	return domain.KeystatsYearParent{
		FinancialYearGroups:    toYearGroupsDomain(p.FinancialYearGroups),
		FinancialYearGroupsUSD: toYearGroupsDomain(p.FinancialYearGroupsUSD),
	}
}

func toYearGroupsDomain(in []stockbit.KeystatsYearGroup) []domain.KeystatsYearGroup {
	out := make([]domain.KeystatsYearGroup, 0, len(in))
	for _, g := range in {
		group := domain.KeystatsYearGroup{
			FinancialYearValues: make([]domain.KeystatsYear, 0, len(g.FinancialYearValues)),
		}
		for _, y := range g.FinancialYearValues {
			year := domain.KeystatsYear{
				Year:            y.Year,
				AnnualisedValue: y.AnnualisedValue,
				TTMValue:        y.TTMValue,
				IsNewUpdate:     y.IsNewUpdate,
				Dividend:        y.Dividend,
				PayoutRatio:     y.PayoutRatio,
				DividendYield:   y.DividendYield,
				PeriodValues:    make([]domain.KeystatsPeriod, 0, len(y.PeriodValues)),
			}
			for _, p := range y.PeriodValues {
				year.PeriodValues = append(year.PeriodValues, domain.KeystatsPeriod{
					Period:       p.Period,
					Year:         p.Year,
					QuarterValue: p.QuarterValue,
					IsNewUpdate:  p.IsNewUpdate,
				})
			}
			group.FinancialYearValues = append(group.FinancialYearValues, year)
		}
		out = append(out, group)
	}
	return out
}

func toDividendGroupDomain(d stockbit.KeystatsDividendGroup) domain.KeystatsDividendGroup {
	out := domain.KeystatsDividendGroup{
		FitemID:            d.FitemID,
		DividendYearValues: make([]domain.KeystatsDividendYear, 0, len(d.DividendYearValues)),
	}
	for _, y := range d.DividendYearValues {
		out.DividendYearValues = append(out.DividendYearValues, domain.KeystatsDividendYear{
			Period:      y.Period,
			Dividend:    y.Dividend,
			ExDate:      y.ExDate,
			PaymentDate: y.PaymentDate,
		})
	}
	return out
}

var _ repository.KeystatsRepository = (*KeystatsRepository)(nil)
