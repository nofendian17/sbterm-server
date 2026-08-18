package stockbit

import (
	"context"
	"net/url"
	"strconv"
)

const brokerTopPath = "/order-trade/broker/top"

// BrokerTopResponse is the top broker response: data wraps the covered session
// date and the per-broker list.
type BrokerTopResponse struct {
	Message string        `json:"message"`
	Data    BrokerTopData `json:"data"`
}

type BrokerTopData struct {
	Date BrokerTopDate   `json:"date"`
	List []BrokerTopItem `json:"list"`
}

type BrokerTopDate struct {
	From string `json:"from"`
	To   string `json:"to"`
	Idx  string `json:"idx"`
}

// BrokerTopItem is one broker's totals. The monetary/volume fields are strings
// because upstream serializes them as such (large IDR/unit amounts).
type BrokerTopItem struct {
	Code           string `json:"code"`
	Name           string `json:"name"`
	InvestorType   string `json:"investor_type"`
	TotalValue     string `json:"total_value"`
	NetValue       string `json:"net_value"`
	BuyValue       string `json:"buy_value"`
	SellValue      string `json:"sell_value"`
	TotalVolume    string `json:"total_volume"`
	TotalFrequency string `json:"total_frequency"`
	Group          string `json:"group"`
}

// GetBrokerTop returns the top brokers ranked by the given sort key over a
// period. eodOnly restricts the data to end-of-day sessions. The access token
// is attached automatically.
func (c *Client) GetBrokerTop(ctx context.Context, sort, order, period, marketType string, eodOnly bool) (*BrokerTopResponse, error) {
	q := url.Values{}
	q.Set("sort", sort)
	q.Set("order", order)
	q.Set("period", period)
	q.Set("market_type", marketType)
	q.Set("eod_only", strconv.FormatBool(eodOnly))
	var out BrokerTopResponse
	if err := c.Get(ctx, brokerTopPath, q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
