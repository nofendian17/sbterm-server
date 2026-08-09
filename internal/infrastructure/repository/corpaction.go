package repository

import (
	"context"

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/infrastructure/stockbit"
	"github.com/nofendian17/sbterm-server/internal/repository"
)

// CorpActionRepository fetches corp actions from the Stockbit API.
type CorpActionRepository struct {
	client *stockbit.Client
}

func NewCorpActionRepository(client *stockbit.Client) *CorpActionRepository {
	return &CorpActionRepository{client: client}
}

func (r *CorpActionRepository) GetCorpActions(ctx context.Context, symbol string, limit int) ([]domain.CompanyCorpAction, error) {
	resp, err := r.client.GetCorpActions(ctx, symbol, limit)
	if err != nil {
		return nil, err
	}
	out := make([]domain.CompanyCorpAction, 0, len(resp.Data))
	for _, a := range resp.Data {
		out = append(out, toDomainCorpAction(a))
	}
	return out, nil
}

func toDomainCorpAction(a stockbit.CompanyCorpAction) domain.CompanyCorpAction {
	out := domain.CompanyCorpAction{ActionType: a.ActionType, Raw: a.Raw}
	if a.Rups != nil {
		out.Rups = &domain.RupsInfo{
			CompanyID:          a.Rups.CompanyID,
			CompanySymbol:      a.Rups.CompanySymbol,
			CorpActionActive:   a.Rups.CorpActionActive,
			CompanyName:        a.Rups.CompanyName,
			CompanyIconURL:     a.Rups.CompanyIconURL,
			RupsCreated:        a.Rups.RupsCreated,
			RupsDatahash:       a.Rups.RupsDatahash,
			RupsDate:           a.Rups.RupsDate,
			RupsID:             a.Rups.RupsID,
			RupsTime:           a.Rups.RupsTime,
			RupsIqpAgenda:      a.Rups.RupsIqpAgenda,
			RupsIqpID:          a.Rups.RupsIqpID,
			RupsIqpRecDt:       a.Rups.RupsIqpRecDt,
			RupsIqpRemark:      a.Rups.RupsIqpRemark,
			RupsIqpResult:      a.Rups.RupsIqpResult,
			RupsIqpRevisedDate: a.Rups.RupsIqpRevisedDate,
			RupsIqpType:        a.Rups.RupsIqpType,
			RupsVenue:          a.Rups.RupsVenue,
			RupsEligibleDate:   a.Rups.RupsEligibleDate,
		}
	}
	if a.RightIssue != nil {
		out.RightIssue = &domain.RightIssueInfo{
			CompanyID:                    a.RightIssue.CompanyID,
			CompanySymbol:                a.RightIssue.CompanySymbol,
			CorpActionActive:             a.RightIssue.CorpActionActive,
			RightIssueCompanyID:          a.RightIssue.RightIssueCompanyID,
			RightIssueAdjFactor:          a.RightIssue.RightIssueAdjFactor,
			RightIssueFactor:             a.RightIssue.RightIssueFactor,
			RightIssueCreated:            a.RightIssue.RightIssueCreated,
			RightIssueCumdate:            a.RightIssue.RightIssueCumdate,
			RightIssueExdate:             a.RightIssue.RightIssueExdate,
			RightIssueLastupdate:         a.RightIssue.RightIssueLastupdate,
			RightIssueID:                 a.RightIssue.RightIssueID,
			RightIssueIqpID:              a.RightIssue.RightIssueIqpID,
			RightIssueLock:               a.RightIssue.RightIssueLock,
			RightIssueNew:                a.RightIssue.RightIssueNew,
			RightIssueNewShare:           a.RightIssue.RightIssueNewShare,
			RightIssueOld:                a.RightIssue.RightIssueOld,
			RightIssuePrice:              a.RightIssue.RightIssuePrice,
			RightIssuePriceAdj:           a.RightIssue.RightIssuePriceAdj,
			RightIssuePriceFactor:        a.RightIssue.RightIssuePriceFactor,
			RightIssuePriceFormatted:     a.RightIssue.RightIssuePriceFormatted,
			RightIssueRatio:              a.RightIssue.RightIssueRatio,
			RightIssueRecdate:            a.RightIssue.RightIssueRecdate,
			RightIssueSubdate:            a.RightIssue.RightIssueSubdate,
			RightIssueTradingEnd:         a.RightIssue.RightIssueTradingEnd,
			RightIssueTradingStart:       a.RightIssue.RightIssueTradingStart,
			RightIssueForeignPercentage:  a.RightIssue.RightIssueForeignPercentage,
			RightIssueLocalPercentage:    a.RightIssue.RightIssueLocalPercentage,
			RightIssueNumberOfSecurities: a.RightIssue.RightIssueNumberOfSecurities,
			RightIssueTotal:              a.RightIssue.RightIssueTotal,
			EventNote:                    a.RightIssue.EventNote,
		}
	}
	if a.StockSplit != nil {
		out.StockSplit = &domain.StockSplitInfo{
			CompanyID:            a.StockSplit.CompanyID,
			CompanySymbol:        a.StockSplit.CompanySymbol,
			CorpActionActive:     a.StockSplit.CorpActionActive,
			StockSplitCreated:    a.StockSplit.StockSplitCreated,
			StockSplitCumdate:    a.StockSplit.StockSplitCumdate,
			StockSplitExdate:     a.StockSplit.StockSplitExdate,
			StockSplitFactor:     a.StockSplit.StockSplitFactor,
			StockSplitID:         a.StockSplit.StockSplitID,
			StockSplitIqpID:      a.StockSplit.StockSplitIqpID,
			StockSplitLastupdate: a.StockSplit.StockSplitLastupdate,
			StockSplitLock:       a.StockSplit.StockSplitLock,
			StockSplitNew:        a.StockSplit.StockSplitNew,
			StockSplitNewPrice:   a.StockSplit.StockSplitNewPrice,
			StockSplitNewShare:   a.StockSplit.StockSplitNewShare,
			StockSplitOld:        a.StockSplit.StockSplitOld,
			StockSplitRatio:      a.StockSplit.StockSplitRatio,
			StockSplitRecdate:    a.StockSplit.StockSplitRecdate,
			EventNote:            a.StockSplit.EventNote,
		}
	}
	return out
}

var _ repository.CorpActionRepository = (*CorpActionRepository)(nil)
