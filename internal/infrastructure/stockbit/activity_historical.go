package stockbit

import (
	"context"
	"net/url"
)

const activityHistoricalPath = "/order-trade/broker/activity/historical"

// ActivityHistoricalResponse is the broker activity historical response.
type ActivityHistoricalResponse struct {
	Message string                 `json:"message"`
	Data    ActivityHistoricalData `json:"data"`
}

// ActivityHistoricalData holds the covered date range, the requested filters,
// the per-interval records, pagination, and the net-value summary grouped by
// net_interval.
type ActivityHistoricalData struct {
	DateFrom    string                     `json:"date_from"`
	DateTo      string                     `json:"date_to"`
	Symbols     []string                   `json:"symbols"`
	BrokerCodes []string                   `json:"broker_codes"`
	BrokerName  string                     `json:"broker_name"`
	Records     []ActivityHistoricalRecord `json:"records"`
	Pagination  ActivityHistoricalPaginate `json:"pagination"`
	Summary     ActivityHistoricalSummary  `json:"summary"`
}

// ActivityHistoricalRecord is one interval bucket: the aggregated trade and
// price activity for a date (and optionally a single broker, when the request
// filters by one broker).
type ActivityHistoricalRecord struct {
	Date          string                  `json:"date"`
	BrokerCode    string                  `json:"broker_code"`
	TradeActivity ActivityHistoricalTrade `json:"trade_activity"`
	PriceActivity ActivityHistoricalPrice `json:"price_activity"`
}

// ActivityHistoricalTrade aggregates buy/sell/net values and lot shares.
type ActivityHistoricalTrade struct {
	NetSummary     ActivitySummary        `json:"net_summary"`
	BuySummary     ActivitySummary        `json:"buy_summary"`
	SellSummary    ActivitySummary        `json:"sell_summary"`
	ForeignSummary ActivityForeignSummary `json:"foreign_summary"`
	TotalBuyLot    ActivityLotShare       `json:"total_buy_lot"`
	TotalSellLot   ActivityLotShare       `json:"total_sell_lot"`
}

// ActivitySummary is one value/volume/frequency summary block. Lot and avg
// price can be fractional.
type ActivitySummary struct {
	AveragePrice float64 `json:"avg_price"`
	Frequency    float64 `json:"freq"`
	Lot          float64 `json:"lot"`
	Value        float64 `json:"value"`
}

// ActivityForeignSummary is the foreign buy/sell flow summary.
type ActivityForeignSummary struct {
	ForeignBuy  float64 `json:"foreign_buy"`
	ForeignSell float64 `json:"foreign_sell"`
	NetForeign  float64 `json:"net_foreign"`
}

// ActivityLotShare is one lot side's absolute amount and market share percent.
type ActivityLotShare struct {
	Amount float64 `json:"amount"`
	Pct    float64 `json:"pct"`
}

// ActivityHistoricalPrice is the price move for the interval bucket. ClosePrice
// is a string because upstream serializes it as such.
type ActivityHistoricalPrice struct {
	ClosePrice    string                        `json:"close_price"`
	ReturnSummary ActivityHistoricalPriceReturn `json:"return_summary"`
}

// ActivityHistoricalPriceReturn is the absolute and percent price change.
type ActivityHistoricalPriceReturn struct {
	Amount float64 `json:"amount"`
	Pct    float64 `json:"pct"`
}

// ActivityHistoricalPaginate is the paging info for the records list.
type ActivityHistoricalPaginate struct {
	Page    int  `json:"page"`
	Limit   int  `json:"limit"`
	HasNext bool `json:"has_next"`
	HasPrev bool `json:"has_prev"`
}

// ActivityHistoricalSummary is the net-value summary grouped by the requested
// net_interval (e.g. monthly buckets over the whole range).
type ActivityHistoricalSummary struct {
	GroupType string                           `json:"group_type"`
	Data      []ActivityHistoricalSummaryGroup `json:"data"`
}

// ActivityHistoricalSummaryGroup is one net_interval bucket of the summary.
type ActivityHistoricalSummaryGroup struct {
	DateFrom   string          `json:"date_from"`
	DateTo     string          `json:"date_to"`
	NetSummary ActivitySummary `json:"net_summary"`
}

// GetActivityHistorical returns per-interval broker activity for the requested
// symbols and broker codes over a date range. interval and netInterval take the
// INTERVAL_* enum values; marketBoard and investorType take the BOARD_TYPE_*
// and INVESTOR_TYPE_* enum values. The access token is attached automatically.
func (c *Client) GetActivityHistorical(ctx context.Context, interval, dateFrom, dateTo string, brokerCodes, symbols []string, marketBoard, investorType, netInterval string) (*ActivityHistoricalResponse, error) {
	q := url.Values{}
	q.Set("interval", interval)
	q.Set("date_from", dateFrom)
	q.Set("date_to", dateTo)
	for _, broker := range brokerCodes {
		q.Add("broker_codes", broker)
	}
	for _, symbol := range symbols {
		q.Add("symbols", symbol)
	}
	q.Set("market_board", marketBoard)
	q.Set("investor_type", investorType)
	q.Set("net_interval", netInterval)
	var out ActivityHistoricalResponse
	if err := c.Get(ctx, activityHistoricalPath, q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
