package repository

import (
	"context"
	"errors"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/apps/api/internal/repository"
	"github.com/nofendian17/sbterm/libs/stockbit"
)

// ForeignDomesticRepository fetches foreign-domestic history from the Stockbit API.
type ForeignDomesticRepository struct {
	client *stockbit.Client
}

func NewForeignDomesticRepository(client *stockbit.Client) *ForeignDomesticRepository {
	return &ForeignDomesticRepository{client: client}
}

func (r *ForeignDomesticRepository) GetForeignDomesticHistorical(ctx context.Context, symbol, marketType, period, from, to string) (*domain.ForeignDomesticData, error) {
	resp, err := r.client.GetForeignDomesticHistorical(ctx, symbol, marketType, period, from, to)
	if err != nil {
		var se *stockbit.StatusError
		if errors.As(err, &se) {
			return nil, &domain.UpstreamError{Status: se.Status, Msg: se.Msg, RetryAfter: se.RetryAfter}
		}
		return nil, err
	}
	d := resp.Data
	out := &domain.ForeignDomesticData{
		LastUpdated:     d.LastUpdated,
		From:            d.From,
		To:              d.To,
		HistoricalPrice: make([]domain.ForeignDomesticPricePoint, 0, len(d.HistoricalPrice)),
		HistoricalNet:   make([]domain.ForeignDomesticNetPoint, 0, len(d.HistoricalNet)),
	}
	for _, p := range d.HistoricalPrice {
		out.HistoricalPrice = append(out.HistoricalPrice, domain.ForeignDomesticPricePoint{
			Date:          p.Date,
			DatetimeLabel: p.DatetimeLabel,
			Open:          toForeignDomesticRawFormatted(p.Open),
			High:          toForeignDomesticRawFormatted(p.High),
			Low:           toForeignDomesticRawFormatted(p.Low),
			Close:         toForeignDomesticRawFormatted(p.Close),
		})
	}
	for _, n := range d.HistoricalNet {
		out.HistoricalNet = append(out.HistoricalNet, domain.ForeignDomesticNetPoint{
			Date:                    n.Date,
			DatetimeLabel:           n.DatetimeLabel,
			DatetimeLabelTable:      n.DatetimeLabelTable,
			NetForeign:              toForeignDomesticValue(n.NetForeign),
			ForeignBuy:              toForeignDomesticValue(n.ForeignBuy),
			ForeignSell:             toForeignDomesticValue(n.ForeignSell),
			ForeignFlow:             toForeignDomesticValue(n.ForeignFlow),
			NetLot:                  toForeignDomesticValue(n.NetLot),
			NetFrequency:            toForeignDomesticValue(n.NetFrequency),
			AveragePrice:            toForeignDomesticValue(n.AveragePrice),
			PercentageForeignValue:  toForeignDomesticValue(n.PercentageForeignValue),
			PercentageDomesticValue: toForeignDomesticValue(n.PercentageDomesticValue),
		})
	}
	return out, nil
}

func toForeignDomesticValue(v stockbit.ForeignDomesticValue) domain.ForeignDomesticValue {
	return domain.ForeignDomesticValue{Raw: v.Raw, Formatted: v.Formatted}
}

func toForeignDomesticRawFormatted(v stockbit.ForeignDomesticRawFormatted) domain.RawFormatted {
	return domain.RawFormatted{Raw: v.Raw, Formatted: v.Formatted}
}

var _ repository.ForeignDomesticRepository = (*ForeignDomesticRepository)(nil)
