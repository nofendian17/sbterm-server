package stockbit

import (
	"context"
	"fmt"
	"net/url"
)

const runningTradeChartPath = "/order-trade/running-trade/chart/%s"

// RunningTradeResponse is the running trade chart response.
type RunningTradeResponse struct {
	Data RunningTradeData `json:"data"`
}

type RunningTradeData struct {
	From            string                    `json:"from"`
	To              string                    `json:"to"`
	DataLastUpdated string                    `json:"data_last_updated"`
	PriceChartData  []RunningTradePricePoint  `json:"price_chart_data"`
	BrokerChartData []RunningTradeBrokerGroup `json:"broker_chart_data"`
	DateSessionInfo string                    `json:"date_session_info"`
}

// RunningTradePricePoint is a single point of the price series. open/high/low
// are pointers because upstream sends null on the broker series.
type RunningTradePricePoint struct {
	Date          string             `json:"date"`
	Time          string             `json:"time"`
	Value         RunningTradeValue  `json:"value"`
	DatetimeLabel string             `json:"datetime_label"`
	Open          *RunningTradeValue `json:"open"`
	High          *RunningTradeValue `json:"high"`
	Low           *RunningTradeValue `json:"low"`
}

type RunningTradeValue struct {
	Raw       string `json:"raw"`
	Formatted string `json:"formatted"`
}

// RunningTradeBrokerGroup is one broker chart series (value or volume).
type RunningTradeBrokerGroup struct {
	Type    string                    `json:"type"`
	Brokers []string                  `json:"brokers"`
	Charts  []RunningTradeBrokerChart `json:"charts"`
}

// RunningTradeBrokerChart is a single broker's series within a group.
type RunningTradeBrokerChart struct {
	BrokerCode string                   `json:"broker_code"`
	Chart      []RunningTradeChartPoint `json:"chart"`
}

// RunningTradeChartPoint is a single point of a broker series.
type RunningTradeChartPoint struct {
	Date          string             `json:"date"`
	Time          string             `json:"time"`
	Value         RunningTradeValue  `json:"value"`
	DatetimeLabel string             `json:"datetime_label"`
	Open          *RunningTradeValue `json:"open"`
	High          *RunningTradeValue `json:"high"`
	Low           *RunningTradeValue `json:"low"`
}

// GetRunningTradeChart returns the running trade chart for a symbol. brokerCodes
// selects the brokers whose series are returned; an empty slice makes upstream
// pick its default set. Either a from/to date range or a period enum
// (RT_PERIOD_*) selects the timeframe; when both are given the from/to range
// wins. investorType and marketBoard take the INVESTOR_TYPE_* and BOARD_TYPE_*
// enum values. The access token is attached automatically.
func (c *Client) GetRunningTradeChart(ctx context.Context, symbol string, brokerCodes []string, from, to, investorType, marketBoard, period string) (*RunningTradeResponse, error) {
	q := url.Values{}
	for _, code := range brokerCodes {
		q.Add("broker_code", code)
	}
	if from != "" && to != "" {
		q.Set("from", from)
		q.Set("to", to)
	} else if period != "" {
		q.Set("period", period)
	}
	q.Set("investor_type", investorType)
	q.Set("market_board", marketBoard)
	var out RunningTradeResponse
	if err := c.Get(ctx, fmt.Sprintf(runningTradeChartPath, symbol), q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
