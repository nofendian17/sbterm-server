package repository

import (
	"context"
	"regexp"

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/infrastructure/stockbit"
	"github.com/nofendian17/sbterm-server/internal/repository"
)

// FindataFinancialRepository fetches the structured financial report from the
// Stockbit API.
type FindataFinancialRepository struct {
	client *stockbit.Client
}

func NewFindataFinancialRepository(client *stockbit.Client) *FindataFinancialRepository {
	return &FindataFinancialRepository{client: client}
}

func (r *FindataFinancialRepository) GetFindataFinancial(ctx context.Context, symbol string, dataType, isPercentage, page, reportType, statementType int) (*domain.FindataFinancial, error) {
	resp, err := r.client.GetFindataFinancial(ctx, symbol, dataType, isPercentage, page, reportType, statementType)
	if err != nil {
		return nil, err
	}
	out := &domain.FindataFinancial{
		Currency:        resp.Data.Currency,
		DefaultCurrency: resp.Data.DefaultCurrency,
		RoundingValue:   resp.Data.RoundingValue,
		DataTables: domain.FindataDataTables{
			Periods:      resp.Data.DataTables.Periods,
			Accounts:     mapFindataAccounts(resp.Data.DataTables.Accounts),
			MaxShowLevel: resp.Data.DataTables.MaxShowLevel,
		},
	}
	return out, nil
}

func mapFindataAccounts(in []stockbit.FindataAccount) []domain.FindataAccount {
	out := make([]domain.FindataAccount, 0, len(in))
	for _, a := range in {
		out = append(out, domain.FindataAccount{
			ID:                a.ID,
			Level:             a.Level,
			Name:              cleanFindataName(a.Name),
			Values:            a.Values,
			Accounts:          mapFindataAccounts(a.Accounts),
			IsTotalExist:      a.IsTotalExist,
			IsDefaultExpanded: a.IsDefaultExpanded,
			MaxShowLevel:      a.MaxShowLevel,
		})
	}
	return out
}

// cleanFindataName strips <b>/</b> markup (any casing) the upstream wraps
// around account headers.
func cleanFindataName(s string) string {
	re := regexp.MustCompile(`(?i)</?b>`)
	return re.ReplaceAllString(s, "")
}

var _ repository.FindataFinancialRepository = (*FindataFinancialRepository)(nil)
