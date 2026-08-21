package stockbit

import (
	"context"
	"net/url"
)

const brokerDistributionPath = "/order-trade/broker/distribution"

// BrokerDistributionResponse is the broker distribution response.
type BrokerDistributionResponse struct {
	Message string                 `json:"message"`
	Data    BrokerDistributionData `json:"data"`
}

type BrokerDistributionData struct {
	DateInfo  string                  `json:"date_info"`
	ByValue   BrokerDistributionSides `json:"by_value"`
	ByVolume  BrokerDistributionSides `json:"by_volume"`
	StartDate string                  `json:"start_date"`
	EndDate   string                  `json:"end_date"`
}

// BrokerDistributionSides holds the top buy/sell brokers. The side selected by
// the request's data_type is populated; the other one stays empty.
type BrokerDistributionSides struct {
	TopBrokerBuy  []BrokerDistributionEntry `json:"top_broker_buy"`
	TopBrokerSell []BrokerDistributionEntry `json:"top_broker_sell"`
}

type BrokerDistributionEntry struct {
	Detail       BrokerDistributionCounterparty   `json:"detail"`
	DistributeTo []BrokerDistributionCounterparty `json:"distribute_to"`
}

type BrokerDistributionCounterparty struct {
	Code   string `json:"code"`
	Type   string `json:"type"` // Lokal / Asing / Pemerintah
	Amount int64  `json:"amount"`
}

// GetBrokerDistribution returns the per-broker buy/sell distribution of one
// symbol. dataType selects whether amounts are value or volume; the sibling
// section comes back empty. Either a single date or a from/to range selects
// the session(s); omitting both returns the latest session. investorType,
// marketBoard and dataType take the INVESTOR_TYPE_*, MARKET_TYPE_* and
// BROKER_DISTRIBUTION_DATA_TYPE_* enum values. The access token is attached
// automatically.
func (c *Client) GetBrokerDistribution(ctx context.Context, symbol, investorType, marketBoard, dataType, date, from, to string) (*BrokerDistributionResponse, error) {
	q := url.Values{}
	q.Set("symbol", symbol)
	if date != "" {
		q.Set("date", date)
	}
	if from != "" && to != "" {
		q.Set("from", from)
		q.Set("to", to)
	}
	q.Set("investor_type", investorType)
	q.Set("market_board", marketBoard)
	q.Set("data_type", dataType)
	var out BrokerDistributionResponse
	if err := c.Get(ctx, brokerDistributionPath, q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
