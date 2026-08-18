package stockbit

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

const chartPricePath = "/chartbit/%s/price/%s"

// ChartPriceResponse is the chartbit OHLC price response.
type ChartPriceResponse struct {
	Message string         `json:"message"`
	Data    ChartPriceData `json:"data"`
}

// flexFloat64 decodes a JSON number or a numeric string ("63300"). Chartbit
// answers use both shapes: daily bars send volume/frequency as numbers while
// intraday bars send them as strings.
type flexFloat64 float64

func (f *flexFloat64) UnmarshalJSON(b []byte) error {
	if s := strings.Trim(string(b), `"`); s == "" || s == "null" {
		*f = 0
		return nil
	} else if v, err := strconv.ParseFloat(s, 64); err != nil {
		return fmt.Errorf("chartbit: parse number %q: %w", s, err)
	} else {
		*f = flexFloat64(v)
		return nil
	}
}

type ChartPriceData struct {
	AllowDecimal int            `json:"allow_decimal"`
	Chartbit     []ChartbitItem `json:"chartbit"`
}

// ChartbitItem is a single OHLC bar. Daily and intraday responses share the
// OHLCV core but each carries a few fields the other omits; both sets of tags
// live in one struct so the same decoder handles either payload.
type ChartbitItem struct {
	Date             string      `json:"date"`
	Unixdate         int64       `json:"unixdate"`
	Datetime         string      `json:"datetime"`
	UnixTimestamp    string      `json:"unix_timestamp"`
	Open             float64     `json:"open"`
	High             float64     `json:"high"`
	Low              float64     `json:"low"`
	Close            float64     `json:"close"`
	Volume           flexFloat64 `json:"volume"`
	Value            float64     `json:"value"`
	Frequency        flexFloat64 `json:"frequency"`
	ForeignBuy       float64     `json:"foreignbuy"`
	ForeignSell      float64     `json:"foreignsell"`
	ForeignFlow      float64     `json:"foreignflow"`
	SoxClose         float64     `json:"soxclose"`
	Dividend         float64     `json:"dividend"`
	ShareOutstanding float64     `json:"shareoutstanding"`
	FreqAnalyzer     float64     `json:"freq_analyzer"`
	Lot              float64     `json:"lot"`
	ForeignBuyToday  float64     `json:"foreign_buy"`
	ForeignSellToday float64     `json:"foreign_sell"`
	Symbol           string      `json:"symbol"`
}

// GetChartPrice returns OHLC price bars for a symbol. timeframe is either
// "daily" (from/to as YYYY-MM-DD) or "intraday" (from/to as Unix seconds; add
// minutes_multiplier for hourly aggregation via WithHeader-free query below).
// The access token is attached automatically.
func (c *Client) GetChartPrice(ctx context.Context, symbol, timeframe, from, to string, limit int) (*ChartPriceResponse, error) {
	q := url.Values{}
	q.Set("from", from)
	q.Set("to", to)
	q.Set("limit", strconv.Itoa(limit))
	var out ChartPriceResponse
	if err := c.Get(ctx, fmt.Sprintf(chartPricePath, symbol, timeframe), q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
