package repository

import (
	"context"
	"errors"

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/infrastructure/stockbit"
	"github.com/nofendian17/sbterm-server/internal/repository"
)

// ActivityRepository fetches broker activity chart and transaction data from the
// Stockbit API.
type ActivityRepository struct {
	client *stockbit.Client
}

func NewActivityRepository(client *stockbit.Client) *ActivityRepository {
	return &ActivityRepository{client: client}
}

func (r *ActivityRepository) GetActivityChart(ctx context.Context, symbols, brokersCode []string, from, to, period, investorType, marketBoard string) (*domain.ActivityChartData, error) {
	resp, err := r.client.GetActivityChart(ctx, symbols, brokersCode, from, to, period, investorType, marketBoard)
	if err != nil {
		var se *stockbit.StatusError
		if errors.As(err, &se) {
			return nil, &domain.UpstreamError{Status: se.Status, Msg: se.Msg}
		}
		return nil, err
	}
	d := resp.Data
	out := &domain.ActivityChartData{
		From:            d.From,
		To:              d.To,
		DataLastUpdated: d.DataLastUpdated,
		ChartData:       make([]domain.ActivityChartGroup, 0, len(d.ChartData)),
		DateSessionInfo: d.DateSessionInfo,
		BrokerCode:      d.BrokerCode,
		BrokerName:      d.BrokerName,
	}
	for _, g := range d.ChartData {
		group := domain.ActivityChartGroup{
			Type:    g.Type,
			Symbols: g.Symbols,
			Charts:  make([]domain.ActivityChartSeries, 0, len(g.Charts)),
		}
		for _, ch := range g.Charts {
			series := domain.ActivityChartSeries{
				Symbol: ch.Symbol,
				Chart:  make([]domain.ActivityChartPoint, 0, len(ch.Chart)),
			}
			for _, p := range ch.Chart {
				series.Chart = append(series.Chart, domain.ActivityChartPoint{
					Date:          p.Date,
					Time:          p.Time,
					Value:         domain.RawFormatted{Raw: p.Value.Raw, Formatted: p.Value.Formatted},
					DatetimeLabel: p.DatetimeLabel,
				})
			}
			group.Charts = append(group.Charts, series)
		}
		out.ChartData = append(out.ChartData, group)
	}
	return out, nil
}

func (r *ActivityRepository) GetActivity(ctx context.Context, brokerCode []string, transactionType, investorType, marketBoard string, limit, page int, from, to, netValPeriod string) (*domain.ActivityData, error) {
	resp, err := r.client.GetActivity(ctx, brokerCode, transactionType, investorType, marketBoard, limit, page, from, to, netValPeriod)
	if err != nil {
		var se *stockbit.StatusError
		if errors.As(err, &se) {
			return nil, &domain.UpstreamError{Status: se.Status, Msg: se.Msg}
		}
		return nil, err
	}
	d := resp.Data
	ba := d.BrokerActivityTransaction
	out := &domain.ActivityData{
		BrokerActivityTransaction: domain.BrokerActivityTransaction{
			BrokersBuy:  toBrokerActivities(ba.BrokersBuy),
			BrokersSell: toBrokerActivities(ba.BrokersSell),
		},
		From:       d.From,
		To:         d.To,
		BrokerCode: d.BrokerCode,
		BrokerName: d.BrokerName,
	}
	return out, nil
}

func toBrokerActivities(in []stockbit.ActivityBrokerActivity) []domain.BrokerActivity {
	out := make([]domain.BrokerActivity, 0, len(in))
	for _, b := range in {
		out = append(out, domain.BrokerActivity{
			StockCode:    b.StockCode,
			BrokerCode:   b.BrokerCode,
			Type:         b.Type,
			Date:         b.Date,
			Value:        b.Value,
			Lot:          b.Lot,
			AveragePrice: b.AveragePrice,
			Frequency:    b.Frequency,
			CompanyDetail: domain.ActivityCompanyDetail{
				IconURL:    b.CompanyDetail.IconURL,
				CorpAction: domain.ActivityCorpAction{Active: b.CompanyDetail.CorpAction.Active, Icon: b.CompanyDetail.CorpAction.Icon, Text: b.CompanyDetail.CorpAction.Text},
				Notation:   toActivityNotations(b.CompanyDetail.Notation),
			},
			NetValueTrend: toNetValueTrend(b.NetValueTrend),
		})
	}
	return out
}

func (r *ActivityRepository) GetActivityHistorical(ctx context.Context, interval, dateFrom, dateTo string, brokerCodes, symbols []string, marketBoard, investorType, netInterval string) (*domain.ActivityHistoricalData, error) {
	resp, err := r.client.GetActivityHistorical(ctx, interval, dateFrom, dateTo, brokerCodes, symbols, marketBoard, investorType, netInterval)
	if err != nil {
		var se *stockbit.StatusError
		if errors.As(err, &se) {
			return nil, &domain.UpstreamError{Status: se.Status, Msg: se.Msg}
		}
		return nil, err
	}
	d := resp.Data
	out := &domain.ActivityHistoricalData{
		DateFrom:    d.DateFrom,
		DateTo:      d.DateTo,
		Symbols:     d.Symbols,
		BrokerCodes: d.BrokerCodes,
		BrokerName:  d.BrokerName,
		Records:     make([]domain.ActivityHistoricalRecord, 0, len(d.Records)),
		Pagination: domain.ActivityHistoricalPaginate{
			Page:    d.Pagination.Page,
			Limit:   d.Pagination.Limit,
			HasNext: d.Pagination.HasNext,
			HasPrev: d.Pagination.HasPrev,
		},
		Summary: domain.ActivityHistoricalSummary{
			GroupType: d.Summary.GroupType,
			Data:      make([]domain.ActivityHistoricalSummaryGroup, 0, len(d.Summary.Data)),
		},
	}
	for _, r := range d.Records {
		out.Records = append(out.Records, domain.ActivityHistoricalRecord{
			Date:       r.Date,
			BrokerCode: r.BrokerCode,
			TradeActivity: domain.ActivityHistoricalTrade{
				NetSummary:     toActivitySummary(r.TradeActivity.NetSummary),
				BuySummary:     toActivitySummary(r.TradeActivity.BuySummary),
				SellSummary:    toActivitySummary(r.TradeActivity.SellSummary),
				ForeignSummary: toActivityForeignSummary(r.TradeActivity.ForeignSummary),
				TotalBuyLot:    toActivityLotShare(r.TradeActivity.TotalBuyLot),
				TotalSellLot:   toActivityLotShare(r.TradeActivity.TotalSellLot),
			},
			PriceActivity: domain.ActivityHistoricalPrice{
				ClosePrice: r.PriceActivity.ClosePrice,
				ReturnSummary: domain.ActivityHistoricalPriceReturn{
					Amount: r.PriceActivity.ReturnSummary.Amount,
					Pct:    r.PriceActivity.ReturnSummary.Pct,
				},
			},
		})
	}
	for _, g := range d.Summary.Data {
		out.Summary.Data = append(out.Summary.Data, domain.ActivityHistoricalSummaryGroup{
			DateFrom:   g.DateFrom,
			DateTo:     g.DateTo,
			NetSummary: toActivitySummary(g.NetSummary),
		})
	}
	return out, nil
}

func toActivitySummary(in stockbit.ActivitySummary) domain.ActivitySummary {
	return domain.ActivitySummary{
		AveragePrice: in.AveragePrice,
		Frequency:    in.Frequency,
		Lot:          in.Lot,
		Value:        in.Value,
	}
}

func toActivityForeignSummary(in stockbit.ActivityForeignSummary) domain.ActivityForeignSummary {
	return domain.ActivityForeignSummary{
		ForeignBuy:  in.ForeignBuy,
		ForeignSell: in.ForeignSell,
		NetForeign:  in.NetForeign,
	}
}

func toActivityLotShare(in stockbit.ActivityLotShare) domain.ActivityLotShare {
	return domain.ActivityLotShare{
		Amount: in.Amount,
		Pct:    in.Pct,
	}
}

func toActivityNotations(in []stockbit.ActivityNotation) []domain.ActivityNotation {
	out := make([]domain.ActivityNotation, 0, len(in))
	for _, n := range in {
		out = append(out, domain.ActivityNotation{
			NotationCode: n.NotationCode,
			NotationDesc: n.NotationDesc,
			IconURL:      domain.ActivityNotationIcon{LightMode: n.IconURL.LightMode, DarkMode: n.IconURL.DarkMode},
		})
	}
	return out
}

func toNetValueTrend(in []stockbit.ActivityNetValueTrend) []domain.ActivityNetValueTrend {
	out := make([]domain.ActivityNetValueTrend, 0, len(in))
	for _, n := range in {
		out = append(out, domain.ActivityNetValueTrend{
			Date:  n.Date,
			NVal:  n.NVal,
			NVol:  n.NVol,
			NFreq: n.NFreq,
		})
	}
	return out
}

var _ repository.ActivityRepository = (*ActivityRepository)(nil)
