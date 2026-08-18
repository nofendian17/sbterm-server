package stockbit

import (
	"context"
	"net/url"
	"strconv"
)

const activityChartPath = "/order-trade/broker/activity-chart"

// ActivityChartDataResponse is the activity chart response.
type ActivityChartDataResponse struct {
	Data ActivityChartData `json:"data"`
}

type ActivityChartData struct {
	From            string               `json:"from"`
	To              string               `json:"to"`
	DataLastUpdated string               `json:"data_last_updated"`
	ChartData       []ActivityChartGroup `json:"chart_data"`
	DateSessionInfo string               `json:"date_session_info"`
	BrokerCode      []string             `json:"broker_code"`
	BrokerName      string               `json:"broker_name"`
}

type ActivityChartGroup struct {
	Type    string                `json:"type"`
	Symbols []string              `json:"symbols"`
	Charts  []ActivityChartSeries `json:"charts"`
}

type ActivityChartSeries struct {
	Symbol string               `json:"symbol"`
	Chart  []ActivityChartPoint `json:"chart"`
}

type ActivityChartPoint struct {
	Date          string            `json:"date"`
	Time          string            `json:"time"`
	Value         RunningTradeValue `json:"value"`
	DatetimeLabel string            `json:"datetime_label"`
}

// GetActivityChart returns the broker activity chart for the requested symbols
// and broker codes. symbols and brokersCode select the series to return; empty
// slices make upstream pick its default set. Either a from/to date range or a
// period enum (RT_PERIOD_*) selects the timeframe; when both are given the
// from/to range wins. investorType and marketBoard take the INVESTOR_TYPE_* and
// BOARD_TYPE_* enum values. The access token is attached automatically.
func (c *Client) GetActivityChart(ctx context.Context, symbols, brokersCode []string, from, to, period, investorType, marketBoard string) (*ActivityChartDataResponse, error) {
	q := url.Values{}
	for _, symbol := range symbols {
		q.Add("symbols", symbol)
	}
	for _, broker := range brokersCode {
		q.Add("brokers_code", broker)
	}
	if from != "" && to != "" {
		q.Set("from", from)
		q.Set("to", to)
	} else if period != "" {
		q.Set("period", period)
	}
	q.Set("investor_type", investorType)
	q.Set("market_board", marketBoard)
	var out ActivityChartDataResponse
	if err := c.Get(ctx, activityChartPath, q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

const activityPath = "/order-trade/broker/activity"

// ActivityDataResponse is the broker activity transaction response.
type ActivityDataResponse struct {
	Data ActivityData `json:"data"`
}

type ActivityData struct {
	BrokerActivityTransaction ActivityBrokerTransaction `json:"broker_activity_transaction"`
	From                      string                    `json:"from"`
	To                        string                    `json:"to"`
	BrokerCode                string                    `json:"broker_code"`
	BrokerName                string                    `json:"broker_name"`
}

type ActivityBrokerTransaction struct {
	BrokersBuy  []ActivityBrokerActivity `json:"brokers_buy"`
	BrokersSell []ActivityBrokerActivity `json:"brokers_sell"`
}

// ActivityBrokerActivity is one flat trading row (a broker's trade on a stock
// on a date). Value/lot/freq are numbers upstream serializes as such; lot and
// avg_price can be fractional.
type ActivityBrokerActivity struct {
	StockCode     string                  `json:"stock_code"`
	BrokerCode    string                  `json:"broker_code"`
	Type          string                  `json:"type"`
	Date          string                  `json:"date"`
	Value         float64                 `json:"value"`
	Lot           float64                 `json:"lot"`
	AveragePrice  float64                 `json:"avg_price"`
	Frequency     float64                 `json:"freq"`
	CompanyDetail ActivityCompanyDetail   `json:"company_detail"`
	NetValueTrend []ActivityNetValueTrend `json:"nval_trend"`
}

type ActivityCompanyDetail struct {
	IconURL    string             `json:"icon_url"`
	CorpAction ActivityCorpAction `json:"corpaction"`
	Notation   []ActivityNotation `json:"notation"`
}

type ActivityNotation struct {
	NotationCode string               `json:"notation_code"`
	NotationDesc string               `json:"notation_desc"`
	IconURL      ActivityNotationIcon `json:"icon_url"`
}

type ActivityNotationIcon struct {
	LightMode string `json:"light_mode"`
	DarkMode  string `json:"dark_mode"`
}

type ActivityCorpAction struct {
	Active bool   `json:"active"`
	Icon   string `json:"icon"`
	Text   string `json:"text"`
}

type ActivityNetValueTrend struct {
	Date  string  `json:"date"`
	NVal  float64 `json:"nval"`
	NVol  float64 `json:"nvol"`
	NFreq float64 `json:"nfreq"`
}

// GetActivity returns the broker activity transaction rows for broker codes.
// brokerCode selects the brokers; transactionType, investorType, and
// marketBoard take the TRANSACTION_TYPE_*, INVESTOR_TYPE_*, and MARKET_TYPE_*
// enum values. limit/page paginate the rows, from/to select the date range, and
// netValPeriod takes the NET_VAL_PERIOD_* enum. The access token is attached
// automatically.
func (c *Client) GetActivity(ctx context.Context, brokerCode []string, transactionType, investorType, marketBoard string, limit, page int, from, to, netValPeriod string) (*ActivityDataResponse, error) {
	q := url.Values{}
	for _, broker := range brokerCode {
		q.Add("broker_code", broker)
	}
	if transactionType != "" {
		q.Set("transaction_type", transactionType)
	}
	if investorType != "" {
		q.Set("investor_type", investorType)
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	if marketBoard != "" {
		q.Set("market_board", marketBoard)
	}
	if page > 0 {
		q.Set("page", strconv.Itoa(page))
	}
	if from != "" && to != "" {
		q.Set("from", from)
		q.Set("to", to)
	}
	if netValPeriod != "" {
		q.Set("net_val_period", netValPeriod)
	}
	var out ActivityDataResponse
	if err := c.Get(ctx, activityPath, q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
