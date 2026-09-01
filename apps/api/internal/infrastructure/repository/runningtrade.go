package repository

import (
	"context"
	"errors"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/repository"
	"github.com/nofendian17/sbterm/libs/stockbit"
)

// RunningTradeRepository fetches running trade charts from the Stockbit API.
type RunningTradeRepository struct {
	client *stockbit.Client
}

func NewRunningTradeRepository(client *stockbit.Client) *RunningTradeRepository {
	return &RunningTradeRepository{client: client}
}

func (r *RunningTradeRepository) GetRunningTradeChart(ctx context.Context, symbol string, brokerCodes []string, from, to, investorType, marketBoard, period string) (*domain.RunningTradeData, error) {
	resp, err := r.client.GetRunningTradeChart(ctx, symbol, brokerCodes, from, to, investorType, marketBoard, period)
	if err != nil {
		// Translate the client's typed status error into a domain error so
		// delivery handlers can map upstream 4xx responses (e.g. 400 for a
		// date whose session has no data yet) to a client-facing status.
		var se *stockbit.StatusError
		if errors.As(err, &se) {
			return nil, &domain.UpstreamError{Status: se.Status, Msg: se.Msg, RetryAfter: se.RetryAfter}
		}
		return nil, err
	}
	d := resp.Data
	out := &domain.RunningTradeData{
		From:            d.From,
		To:              d.To,
		DataLastUpdated: d.DataLastUpdated,
		PriceChartData:  make([]domain.RunningTradePricePoint, 0, len(d.PriceChartData)),
		BrokerChartData: make([]domain.RunningTradeBrokerGroup, 0, len(d.BrokerChartData)),
		DateSessionInfo: d.DateSessionInfo,
	}
	for _, p := range d.PriceChartData {
		out.PriceChartData = append(out.PriceChartData, domain.RunningTradePricePoint{
			Date:          p.Date,
			Time:          p.Time,
			Value:         toRawFormatted(p.Value),
			DatetimeLabel: p.DatetimeLabel,
			Open:          toRawFormattedPtr(p.Open),
			High:          toRawFormattedPtr(p.High),
			Low:           toRawFormattedPtr(p.Low),
		})
	}
	for _, g := range d.BrokerChartData {
		group := domain.RunningTradeBrokerGroup{
			Type:    g.Type,
			Brokers: g.Brokers,
			Charts:  make([]domain.RunningTradeBrokerChart, 0, len(g.Charts)),
		}
		for _, ch := range g.Charts {
			group.Charts = append(group.Charts, domain.RunningTradeBrokerChart{
				BrokerCode: ch.BrokerCode,
				Chart:      toRunningTradeChartPoints(ch.Chart),
			})
		}
		out.BrokerChartData = append(out.BrokerChartData, group)
	}
	return out, nil
}

func (r *RunningTradeRepository) GetRunningTrade(ctx context.Context, symbol, sort, orderBy, date string, limit int, tradeNumber int64) (*domain.RunningTradeFeed, error) {
	resp, err := r.client.GetRunningTrade(ctx, symbol, sort, orderBy, date, limit, tradeNumber)
	if err != nil {
		var se *stockbit.StatusError
		if errors.As(err, &se) {
			return nil, &domain.UpstreamError{Status: se.Status, Msg: se.Msg, RetryAfter: se.RetryAfter}
		}
		return nil, err
	}
	d := resp.Data
	out := &domain.RunningTradeFeed{
		IsOpenMarket: d.IsOpenMarket,
		RunningTrade: make([]domain.RunningTradeFeedItem, 0, len(d.RunningTrade)),
	}
	for _, t := range d.RunningTrade {
		out.RunningTrade = append(out.RunningTrade, domain.RunningTradeFeedItem{
			ID:               t.ID,
			Time:             t.Time,
			Action:           t.Action,
			Code:             t.Code,
			Price:            t.Price,
			Change:           t.Change,
			Lot:              t.Lot,
			IsBrokerExists:   t.IsBrokerExists,
			Buyer:            t.Buyer,
			Seller:           t.Seller,
			TradeNumber:      t.TradeNumber,
			BuyerType:        t.BuyerType,
			SellerType:       t.SellerType,
			MarketBoard:      t.MarketBoard,
			BuyOrderNumber:   t.BuyOrderNumber,
			SellOrderNumber:  t.SellOrderNumber,
			GroupOrderNumber: t.GroupOrderNumber,
			Value:            domain.RunningTradeFeedValue{Raw: t.Value.Raw, Formatted: t.Value.Formatted},
		})
	}
	return out, nil
}

func (r *RunningTradeRepository) GetRunningTradeGroup(ctx context.Context, symbol, sort, orderBy, date, marketBoard string, limit int, cursor int64) (*domain.RunningTradeGroupFeed, error) {
	resp, err := r.client.GetRunningTradeGroup(ctx, symbol, sort, orderBy, date, marketBoard, limit, cursor)
	if err != nil {
		var se *stockbit.StatusError
		if errors.As(err, &se) {
			return nil, &domain.UpstreamError{Status: se.Status, Msg: se.Msg, RetryAfter: se.RetryAfter}
		}
		return nil, err
	}
	d := resp.Data
	out := &domain.RunningTradeGroupFeed{
		Date:        d.Date,
		SingleOrder: d.SingleOrder,
		Total: domain.RunningTradeGroupTotal{
			Value:     domain.RawFormatted{Raw: d.Total.Value.Raw.String(), Formatted: d.Total.Value.Formatted},
			Lot:       domain.RawFormatted{Raw: d.Total.Lot.Raw.String(), Formatted: d.Total.Lot.Formatted},
			Frequency: domain.RawFormatted{Raw: d.Total.Frequency.Raw.String(), Formatted: d.Total.Frequency.Formatted},
		},
		RunningTradeGroup: make([]domain.RunningTradeGroupItem, 0, len(d.RunningTradeGroup)),
	}
	for _, t := range d.RunningTradeGroup {
		item := domain.RunningTradeGroupItem{
			ID:             t.ID,
			OrderNumber:    t.OrderNumber,
			Action:         t.Action,
			GroupAction:    t.GroupAction,
			Time:           t.Time,
			TradeNumber:    t.TradeNumber,
			Code:           t.Code,
			MarketBoard:    t.MarketBoard,
			Price:          domain.RawFormatted{Raw: t.Price.Raw.String(), Formatted: t.Price.Formatted},
			Change:         domain.RawFormatted{Raw: t.Change.Raw.String(), Formatted: t.Change.Formatted},
			Lot:            domain.RawFormatted{Raw: t.Lot.Raw.String(), Formatted: t.Lot.Formatted},
			Freq:           domain.RawFormatted{Raw: t.Freq.Raw.String(), Formatted: t.Freq.Formatted},
			IsBrokerExists: t.IsBrokerExists,
			Value:          domain.RawFormatted{Raw: t.Value.Raw.String(), Formatted: t.Value.Formatted},
			Buyer:          make([]domain.RunningTradeGroupBroker, 0, len(t.Buyer)),
			Seller:         make([]domain.RunningTradeGroupBroker, 0, len(t.Seller)),
		}
		for _, b := range t.Buyer {
			item.Buyer = append(item.Buyer, domain.RunningTradeGroupBroker{BrokerCode: b.BrokerCode, BrokerType: b.BrokerType})
		}
		for _, s := range t.Seller {
			item.Seller = append(item.Seller, domain.RunningTradeGroupBroker{BrokerCode: s.BrokerCode, BrokerType: s.BrokerType})
		}
		out.RunningTradeGroup = append(out.RunningTradeGroup, item)
	}
	return out, nil
}

func toRunningTradeChartPoints(points []stockbit.RunningTradeChartPoint) []domain.RunningTradeChartPoint {
	out := make([]domain.RunningTradeChartPoint, 0, len(points))
	for _, p := range points {
		out = append(out, domain.RunningTradeChartPoint{
			Date:          p.Date,
			Time:          p.Time,
			Value:         toRawFormatted(p.Value),
			DatetimeLabel: p.DatetimeLabel,
			Open:          toRawFormattedPtr(p.Open),
			High:          toRawFormattedPtr(p.High),
			Low:           toRawFormattedPtr(p.Low),
		})
	}
	return out
}

func toRawFormatted(v stockbit.RunningTradeValue) domain.RawFormatted {
	return domain.RawFormatted{Raw: v.Raw, Formatted: v.Formatted}
}

func toRawFormattedPtr(v *stockbit.RunningTradeValue) *domain.RawFormatted {
	if v == nil {
		return nil
	}
	rf := toRawFormatted(*v)
	return &rf
}

var _ repository.RunningTradeRepository = (*RunningTradeRepository)(nil)
