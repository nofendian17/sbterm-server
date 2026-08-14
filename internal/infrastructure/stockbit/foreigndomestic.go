package stockbit

import (
	"context"
	"encoding/json"
	"net/url"
)

const foreignDomesticPath = "/order-trade/foreign-domestic/historical"

// ForeignDomesticResponse is the foreign-domestic historical response.
type ForeignDomesticResponse struct {
	Message string              `json:"message"`
	Data    ForeignDomesticData `json:"data"`
}

type ForeignDomesticData struct {
	HistoricalPrice []ForeignDomesticPricePoint `json:"historical_price"`
	HistoricalNet   []ForeignDomesticNetPoint   `json:"historical_net"`
	LastUpdated     string                      `json:"last_updated"`
	From            string                      `json:"from"`
	To              string                      `json:"to"`
}

// ForeignDomesticPricePoint is one OHLC bar of the price series; raw values are
// display strings upstream.
type ForeignDomesticPricePoint struct {
	Date          string                      `json:"date"`
	DatetimeLabel string                      `json:"datetime_label"`
	Open          ForeignDomesticRawFormatted `json:"open"`
	High          ForeignDomesticRawFormatted `json:"high"`
	Low           ForeignDomesticRawFormatted `json:"low"`
	Close         ForeignDomesticRawFormatted `json:"close"`
}

// ForeignDomesticNetPoint is one day's foreign-buy/sell aggregates.
type ForeignDomesticNetPoint struct {
	Date                    string               `json:"date"`
	DatetimeLabel           string               `json:"datetime_label"`
	DatetimeLabelTable      string               `json:"datetime_label_table"`
	NetForeign              ForeignDomesticValue `json:"net_foreign"`
	ForeignBuy              ForeignDomesticValue `json:"foreign_buy"`
	ForeignSell             ForeignDomesticValue `json:"foreign_sell"`
	ForeignFlow             ForeignDomesticValue `json:"foreign_flow"`
	NetLot                  ForeignDomesticValue `json:"net_lot"`
	NetFrequency            ForeignDomesticValue `json:"net_frequency"`
	AveragePrice            ForeignDomesticValue `json:"average_price"`
	PercentageForeignValue  ForeignDomesticValue `json:"percentage_foreign_value"`
	PercentageDomesticValue ForeignDomesticValue `json:"percentage_domestic_value"`
}

type ForeignDomesticRawFormatted struct {
	Raw       string `json:"raw"`
	Formatted string `json:"formatted"`
}

// ForeignDomesticValue is a foreign-domestic aggregate: upstream raw is a
// number for most fields but a string for net_frequency and a float for the
// percentages, so keep it as json.Number.
type ForeignDomesticValue struct {
	Raw       json.Number `json:"raw"`
	Formatted string      `json:"formatted"`
}

// GetForeignDomesticHistorical returns the foreign/domestic buy-sell history
// for a symbol. marketType accepts MARKET_TYPE_ALL (the only value upstream
// accepts); period takes the TB_PERIOD_* enums. When both from and to are set
// they select a date range (which wins over period); otherwise period is used.
// The access token is attached automatically; the endpoint works with the
// default mobile headers.
func (c *Client) GetForeignDomesticHistorical(ctx context.Context, symbol, marketType, period, from, to string) (*ForeignDomesticResponse, error) {
	q := url.Values{}
	q.Set("symbols", symbol)
	q.Set("market_type", marketType)
	if from != "" && to != "" {
		q.Set("from", from)
		q.Set("to", to)
	} else {
		q.Set("period", period)
	}
	var out ForeignDomesticResponse
	if err := c.Get(ctx, foreignDomesticPath, q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
