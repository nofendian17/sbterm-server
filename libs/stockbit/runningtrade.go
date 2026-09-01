package stockbit

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

const (
	runningTradeChartPath = "/order-trade/running-trade/chart/%s"
	runningTradePath      = "/order-trade/running-trade"
	runningTradeGroupPath = "/order-trade/running-trade/group"
)

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

// RunningTradeFeedResponse is the running trade feed response.
type RunningTradeFeedResponse struct {
	Message string               `json:"message"`
	Data    RunningTradeFeedData `json:"data"`
}

type RunningTradeFeedData struct {
	IsOpenMarket bool                   `json:"is_open_market"`
	RunningTrade []RunningTradeFeedItem `json:"running_trade"`
}

// RunningTradeFeedItem is one executed trade in the feed. Lot/price/change are
// display-formatted strings; value carries a numeric raw and a formatted string.
type RunningTradeFeedItem struct {
	ID               string                `json:"id"`
	Time             string                `json:"time"`
	Action           string                `json:"action"`
	Code             string                `json:"code"`
	Price            string                `json:"price"`
	Change           string                `json:"change"`
	Lot              string                `json:"lot"`
	IsBrokerExists   bool                  `json:"is_broker_exists"`
	Buyer            string                `json:"buyer"`
	Seller           string                `json:"seller"`
	TradeNumber      string                `json:"trade_number"`
	BuyerType        string                `json:"buyer_type"`
	SellerType       string                `json:"seller_type"`
	MarketBoard      string                `json:"market_board"`
	BuyOrderNumber   string                `json:"buy_order_number"`
	SellOrderNumber  string                `json:"sell_order_number"`
	GroupOrderNumber string                `json:"group_order_number"`
	Value            RunningTradeFeedValue `json:"value"`
}

// RunningTradeFeedValue is a running trade value: upstream sends raw as a JSON
// number (kept as json.Number so either 630000 or "630000" decodes) and
// formatted as a display string.
type RunningTradeFeedValue struct {
	Raw       json.Number `json:"raw"`
	Formatted string      `json:"formatted"`
}

// GetRunningTrade returns the running trade feed for a single symbol. sort
// (ASC/DESC) and orderBy (RUNNING_TRADE_ORDER_BY_*) select the ordering;
// tradeNumber is a cursor for paging (pass the last row's trade_number to fetch
// the next page); date selects a session (YYYY-MM-DD) — empty lets upstream
// fall back to the most recent data. The access token is attached automatically.
func (c *Client) GetRunningTrade(ctx context.Context, symbol, sort, orderBy, date string, limit int, tradeNumber int64) (*RunningTradeFeedResponse, error) {
	q := url.Values{}
	q.Set("symbols[]", symbol)
	if sort != "" {
		q.Set("sort", sort)
	}
	if orderBy != "" {
		q.Set("order_by", orderBy)
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	if tradeNumber > 0 {
		q.Set("trade_number", strconv.FormatInt(tradeNumber, 10))
	}
	if date != "" {
		q.Set("date", date)
	}
	var out RunningTradeFeedResponse
	if err := c.Get(ctx, runningTradePath, q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// RunningTradeGroupResponse is the running trade group feed response.
type RunningTradeGroupResponse struct {
	Message string                `json:"message"`
	Data    RunningTradeGroupData `json:"data"`
}

// RunningTradeGroupData holds the grouped running trade data.
type RunningTradeGroupData struct {
	Total             RunningTradeGroupTotal  `json:"total"`
	RunningTradeGroup []RunningTradeGroupItem `json:"running_trade_group"`
	Date              string                  `json:"date"`
	SingleOrder       bool                    `json:"single_order"`
}

// RunningTradeGroupTotal is the aggregated totals for the group feed.
type RunningTradeGroupTotal struct {
	Value     RunningTradeGroupValue `json:"value"`
	Lot       RunningTradeGroupValue `json:"lot"`
	Frequency RunningTradeGroupValue `json:"frequency"`
}

// RunningTradeGroupItem is one grouped trade in the feed.
type RunningTradeGroupItem struct {
	ID             string                    `json:"id"`
	OrderNumber    string                    `json:"order_number"`
	Action         string                    `json:"action"`
	GroupAction    string                    `json:"group_action"`
	Time           string                    `json:"time"`
	TradeNumber    string                    `json:"trade_number"`
	Code           string                    `json:"code"`
	MarketBoard    string                    `json:"market_board"`
	Price          RunningTradeGroupValue    `json:"price"`
	Change         RunningTradeGroupValue    `json:"change"`
	Lot            RunningTradeGroupValue    `json:"lot"`
	Freq           RunningTradeGroupValue    `json:"freq"`
	IsBrokerExists bool                      `json:"is_broker_exists"`
	Buyer          []RunningTradeGroupBroker `json:"buyer"`
	Seller         []RunningTradeGroupBroker `json:"seller"`
	Value          RunningTradeGroupValue    `json:"value"`
}

// RunningTradeGroupBroker is a broker entry in a group item's buyer/seller list.
type RunningTradeGroupBroker struct {
	BrokerCode string `json:"broker_code"`
	BrokerType string `json:"broker_type"`
}

// RunningTradeGroupValue is a value with raw and formatted representations.
type RunningTradeGroupValue struct {
	Raw       json.Number `json:"raw"`
	Formatted string      `json:"formatted"`
}

// GetRunningTradeGroup returns the grouped running trade feed for a symbol.
// sort (ASC/DESC) and orderBy (RUNNING_TRADE_ORDER_BY_*) select the ordering;
// cursor is the id of the last item for paging; date selects a session
// (YYYY-MM-DD) — empty lets upstream fall back to the most recent data;
// marketBoard takes BOARD_TYPE_* values. The access token is attached
// automatically.
func (c *Client) GetRunningTradeGroup(ctx context.Context, symbol, sort, orderBy, date, marketBoard string, limit int, cursor int64) (*RunningTradeGroupResponse, error) {
	q := url.Values{}
	q.Set("symbols", symbol)
	if sort != "" {
		q.Set("sort", sort)
	}
	if orderBy != "" {
		q.Set("order_by", orderBy)
	}
	if marketBoard != "" {
		q.Set("market_board", marketBoard)
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	if cursor > 0 {
		q.Set("cursor", strconv.FormatInt(cursor, 10))
	}
	if date != "" {
		q.Set("date", date)
	}
	var out RunningTradeGroupResponse
	if err := c.Get(ctx, runningTradeGroupPath, q, &out); err != nil {
		return nil, err
	}
	return &out, nil
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
