package repository

import (
	"context"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/repository"
	"github.com/nofendian17/sbterm/libs/stockbit"
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

func (r *CorpActionRepository) GetCorpActionCalendar(ctx context.Context, date string) (*domain.CorpActionCalendar, error) {
	resp, err := r.client.GetCorpActionCalendar(ctx, date)
	if err != nil {
		return nil, err
	}
	d := resp.Data
	return &domain.CorpActionCalendar{
		Bonus:         d.Bonus,
		Dividend:      toDomainDividends(d.Dividend),
		Economic:      d.Economic,
		Ipo:           d.Ipo,
		Pubex:         d.Pubex,
		RightIssue:    toDomainRightIssues(d.RightIssue),
		Rups:          toDomainRupsList(d.Rups),
		StockReverse:  d.StockReverse,
		StockSplit:    toDomainStockSplits(d.StockSplit),
		Tender:        toDomainTenders(d.Tender),
		Warrant:       toDomainWarrants(d.Warrant),
		StockDividend: d.StockDividend,
		Today:         d.Today,
	}, nil
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

func toDomainDividends(in []stockbit.DividendInfo) []domain.DividendInfo {
	out := make([]domain.DividendInfo, 0, len(in))
	for _, d := range in {
		out = append(out, domain.DividendInfo{
			CompanyID:              d.CompanyID,
			CompanySymbol:          d.CompanySymbol,
			CorpActionActive:       d.CorpActionActive,
			DividendCreated:        d.DividendCreated,
			DividendCumdate:        d.DividendCumdate,
			DividendDatahash:       d.DividendDatahash,
			DividendExdate:         d.DividendExdate,
			DividendID:             d.DividendID,
			DividendIqpID:          d.DividendIqpID,
			DividendLastupdate:     d.DividendLastupdate,
			DividendLock:           d.DividendLock,
			DividendPaydate:        d.DividendPaydate,
			DividendRecdate:        d.DividendRecdate,
			DividendValue:          d.DividendValue,
			Lastprice:              d.Lastprice,
			EventNote:              d.EventNote,
			DividendValueFormatted: d.DividendValueFormatted,
			LastpriceFormatted:     d.LastpriceFormatted,
			DividendCurrency:       d.DividendCurrency,
			DividendFiscalYear:     d.DividendFiscalYear,
			DividendValueAdjusted:  d.DividendValueAdjusted,
		})
	}
	return out
}

func toDomainRightIssues(in []stockbit.RightIssueInfo) []domain.RightIssueInfo {
	out := make([]domain.RightIssueInfo, 0, len(in))
	for _, ri := range in {
		out = append(out, domain.RightIssueInfo{
			CompanyID:                    ri.CompanyID,
			CompanySymbol:                ri.CompanySymbol,
			CorpActionActive:             ri.CorpActionActive,
			RightIssueCompanyID:          ri.RightIssueCompanyID,
			RightIssueAdjFactor:          ri.RightIssueAdjFactor,
			RightIssueFactor:             ri.RightIssueFactor,
			RightIssueCreated:            ri.RightIssueCreated,
			RightIssueCumdate:            ri.RightIssueCumdate,
			RightIssueExdate:             ri.RightIssueExdate,
			RightIssueLastupdate:         ri.RightIssueLastupdate,
			RightIssueID:                 ri.RightIssueID,
			RightIssueIqpID:              ri.RightIssueIqpID,
			RightIssueLock:               ri.RightIssueLock,
			RightIssueNew:                ri.RightIssueNew,
			RightIssueNewShare:           ri.RightIssueNewShare,
			RightIssueOld:                ri.RightIssueOld,
			RightIssuePrice:              ri.RightIssuePrice,
			RightIssuePriceAdj:           ri.RightIssuePriceAdj,
			RightIssuePriceFactor:        ri.RightIssuePriceFactor,
			RightIssuePriceFormatted:     ri.RightIssuePriceFormatted,
			RightIssueRatio:              ri.RightIssueRatio,
			RightIssueRecdate:            ri.RightIssueRecdate,
			RightIssueSubdate:            ri.RightIssueSubdate,
			RightIssueTradingEnd:         ri.RightIssueTradingEnd,
			RightIssueTradingStart:       ri.RightIssueTradingStart,
			RightIssueForeignPercentage:  ri.RightIssueForeignPercentage,
			RightIssueLocalPercentage:    ri.RightIssueLocalPercentage,
			RightIssueNumberOfSecurities: ri.RightIssueNumberOfSecurities,
			RightIssueTotal:              ri.RightIssueTotal,
			EventNote:                    ri.EventNote,
		})
	}
	return out
}

func toDomainRupsList(in []stockbit.RupsInfo) []domain.RupsInfo {
	out := make([]domain.RupsInfo, 0, len(in))
	for _, ru := range in {
		out = append(out, domain.RupsInfo{
			CompanyID:          ru.CompanyID,
			CompanySymbol:      ru.CompanySymbol,
			CorpActionActive:   ru.CorpActionActive,
			CompanyName:        ru.CompanyName,
			CompanyIconURL:     ru.CompanyIconURL,
			RupsCreated:        ru.RupsCreated,
			RupsDatahash:       ru.RupsDatahash,
			RupsDate:           ru.RupsDate,
			RupsID:             ru.RupsID,
			RupsTime:           ru.RupsTime,
			RupsIqpAgenda:      ru.RupsIqpAgenda,
			RupsIqpID:          ru.RupsIqpID,
			RupsIqpRecDt:       ru.RupsIqpRecDt,
			RupsIqpRemark:      ru.RupsIqpRemark,
			RupsIqpResult:      ru.RupsIqpResult,
			RupsIqpRevisedDate: ru.RupsIqpRevisedDate,
			RupsIqpType:        ru.RupsIqpType,
			RupsVenue:          ru.RupsVenue,
			RupsEligibleDate:   ru.RupsEligibleDate,
		})
	}
	return out
}

func toDomainStockSplits(in []stockbit.StockSplitInfo) []domain.StockSplitInfo {
	out := make([]domain.StockSplitInfo, 0, len(in))
	for _, ss := range in {
		out = append(out, domain.StockSplitInfo{
			CompanyID:            ss.CompanyID,
			CompanySymbol:        ss.CompanySymbol,
			CorpActionActive:     ss.CorpActionActive,
			StockSplitCreated:    ss.StockSplitCreated,
			StockSplitCumdate:    ss.StockSplitCumdate,
			StockSplitExdate:     ss.StockSplitExdate,
			StockSplitFactor:     ss.StockSplitFactor,
			StockSplitID:         ss.StockSplitID,
			StockSplitIqpID:      ss.StockSplitIqpID,
			StockSplitLastupdate: ss.StockSplitLastupdate,
			StockSplitLock:       ss.StockSplitLock,
			StockSplitNew:        ss.StockSplitNew,
			StockSplitNewPrice:   ss.StockSplitNewPrice,
			StockSplitNewShare:   ss.StockSplitNewShare,
			StockSplitOld:        ss.StockSplitOld,
			StockSplitRatio:      ss.StockSplitRatio,
			StockSplitRecdate:    ss.StockSplitRecdate,
			EventNote:            ss.EventNote,
		})
	}
	return out
}

func toDomainTenders(in []stockbit.TenderInfo) []domain.TenderInfo {
	out := make([]domain.TenderInfo, 0, len(in))
	for _, td := range in {
		out = append(out, domain.TenderInfo{
			CompanyID:            td.CompanyID,
			CompanyName:          td.CompanyName,
			CompanySymbol:        td.CompanySymbol,
			CorpActionActive:     td.CorpActionActive,
			TenderCreated:        td.TenderCreated,
			TenderDatahash:       td.TenderDatahash,
			TenderEnd:            td.TenderEnd,
			TenderID:             td.TenderID,
			TenderPaydate:        td.TenderPaydate,
			TenderPercentage:     td.TenderPercentage,
			TenderPrice:          td.TenderPrice,
			TenderShares:         td.TenderShares,
			TenderStart:          td.TenderStart,
			EventNote:            td.EventNote,
			TenderPriceFormatted: td.TenderPriceFormatted,
		})
	}
	return out
}

func toDomainWarrants(in []stockbit.WarrantInfo) []domain.WarrantInfo {
	out := make([]domain.WarrantInfo, 0, len(in))
	for _, wr := range in {
		out = append(out, domain.WarrantInfo{
			CompanyID:               wr.CompanyID,
			CompanySymbol:           wr.CompanySymbol,
			CorpActionActive:        wr.CorpActionActive,
			WrantExcEnd:             wr.WrantExcEnd,
			WrantExcFrom:            wr.WrantExcFrom,
			WrantExcPrice:           wr.WrantExcPrice,
			WrantID:                 wr.WrantID,
			WrantIqpID:              wr.WrantIqpID,
			WrantLastupdate:         wr.WrantLastupdate,
			WrantSerie:              wr.WrantSerie,
			WrantTotal:              wr.WrantTotal,
			WrantTradingEnd:         wr.WrantTradingEnd,
			WrantTradingFrom:        wr.WrantTradingFrom,
			EventNote:               wr.EventNote,
			WrantExcPriceFormatted:  wr.WrantExcPriceFormatted,
			WrantForeignPercentage:  wr.WrantForeignPercentage,
			WrantLocalPercentage:    wr.WrantLocalPercentage,
			WrantNumberOfSecurities: wr.WrantNumberOfSecurities,
			WrantCompanyID:          wr.WrantCompanyID,
		})
	}
	return out
}

var _ repository.CorpActionRepository = (*CorpActionRepository)(nil)
