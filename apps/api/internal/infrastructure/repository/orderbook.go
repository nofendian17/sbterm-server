package repository

import (
	"context"
	"errors"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/repository"
	"github.com/nofendian17/sbterm/libs/stockbit"
)

// OrderBookRepository fetches order books from the Stockbit API.
type OrderBookRepository struct {
	client *stockbit.Client
}

func NewOrderBookRepository(client *stockbit.Client) *OrderBookRepository {
	return &OrderBookRepository{client: client}
}

func (r *OrderBookRepository) GetOrderBook(ctx context.Context, symbol string) (*domain.OrderBookData, error) {
	resp, err := r.client.GetOrderBook(ctx, symbol)
	if err != nil {
		var se *stockbit.StatusError
		if errors.As(err, &se) {
			return nil, &domain.UpstreamError{Status: se.Status, Msg: se.Msg}
		}
		return nil, err
	}
	d := resp.Data
	out := &domain.OrderBookData{
		Average:                      d.Average,
		Change:                       d.Change,
		Close:                        d.Close,
		Country:                      d.Country,
		Domestic:                     d.Domestic,
		Down:                         d.Down,
		Exchange:                     d.Exchange,
		FBuy:                         d.FBuy,
		FNet:                         d.FNet,
		Foreign:                      d.Foreign,
		Frequency:                    d.Frequency,
		FSell:                        d.FSell,
		High:                         d.High,
		ID:                           d.ID,
		LastPrice:                    d.LastPrice,
		Low:                          d.Low,
		Open:                         d.Open,
		PercentageChange:             d.PercentageChange,
		Previous:                     d.Previous,
		Status:                       d.Status,
		Symbol:                       d.Symbol,
		Symbol2:                      d.Symbol2,
		Symbol3:                      d.Symbol3,
		Tradable:                     d.Tradable,
		Unchanged:                    d.Unchanged,
		Up:                           d.Up,
		Value:                        d.Value,
		Volume:                       d.Volume,
		CorpAction:                   domain.OrderBookCorpAction{Active: d.CorpAction.Active, Icon: d.CorpAction.Icon, Text: d.CorpAction.Text},
		Notation:                     d.Notation,
		UMA:                          d.UMA,
		HasForeignBS:                 d.HasForeignBS,
		IEPIEV:                       d.IEPIEV,
		MarketData:                   toOrderBookMarkets(d.MarketData),
		Name:                         d.Name,
		IconURL:                      d.IconURL,
		ARA:                          domain.OrderBookLimit{Value: d.ARA.Value, Visible: d.ARA.Visible},
		ARB:                          domain.OrderBookLimit{Value: d.ARB.Value, Visible: d.ARB.Visible},
		CompanyType:                  d.CompanyType,
		TotalBidOffer:                domain.OrderBookTotal{Bid: toOrderBookSide(d.TotalBidOffer.Bid), Offer: toOrderBookSide(d.TotalBidOffer.Offer), BidPercent: d.TotalBidOffer.BidPercent},
		NextARA:                      domain.OrderBookLimit{Value: d.NextARA.Value, Visible: d.NextARA.Visible},
		NextARB:                      domain.OrderBookLimit{Value: d.NextARB.Value, Visible: d.NextARB.Visible},
		AutoRejectTimeLeftInSec:      d.AutoRejectTimeLeftInSec,
		AutoRejectEstimation:         d.AutoRejectEstimation,
		OrderbookActiveFeatureMobile: d.OrderbookActiveFeatureMobile,
	}
	out.Bid = make([]domain.OrderBookLevel, 0, len(d.Bid))
	for _, l := range d.Bid {
		out.Bid = append(out.Bid, toOrderBookLevel(l))
	}
	out.Offer = make([]domain.OrderBookLevel, 0, len(d.Offer))
	for _, l := range d.Offer {
		out.Offer = append(out.Offer, toOrderBookLevel(l))
	}
	return out, nil
}

func toOrderBookLevel(l stockbit.OrderBookLevel) domain.OrderBookLevel {
	return domain.OrderBookLevel{Price: l.Price, QueNum: l.QueNum, Volume: l.Volume, ChangePercentage: l.ChangePercentage}
}

func toOrderBookMarkets(in []stockbit.OrderBookMarket) []domain.OrderBookMarket {
	out := make([]domain.OrderBookMarket, 0, len(in))
	for _, m := range in {
		out = append(out, domain.OrderBookMarket{
			Label:     m.Label,
			Frequency: domain.RawFormatted{Raw: m.Frequency.Raw, Formatted: m.Frequency.Formatted},
			Volume:    domain.RawFormatted{Raw: m.Volume.Raw, Formatted: m.Volume.Formatted},
			Value:     domain.RawFormatted{Raw: m.Value.Raw, Formatted: m.Value.Formatted},
		})
	}
	return out
}

func toOrderBookSide(s stockbit.OrderBookSide) domain.OrderBookSide {
	return domain.OrderBookSide{Freq: s.Freq, Lot: s.Lot, RawLot: s.RawLot, RawFreq: s.RawFreq}
}

var _ repository.OrderBookRepository = (*OrderBookRepository)(nil)
