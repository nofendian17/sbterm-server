package stockbit

import (
	"context"
	"fmt"
	"net/url"
)

const indexChartPath = "/charts/%s/daily"

// IndexSummaryResponse is the index chart summary response. data carries the
// per-day summary fields and the price series for the requested interval.
type IndexSummaryResponse struct {
	Data IndexSummaryData `json:"data"`
}

type IndexSummaryData struct {
	Cagr                   string              `json:"cagr"`
	Change                 flexFloat64         `json:"change"`
	Drawdown               string              `json:"drawdown"`
	MarkingPoint           string              `json:"markingpoint"`
	Percentage             string              `json:"percentage"`
	Timeframe              string              `json:"timeframe"`
	XAxisOpt               string              `json:"xaxisopt"`
	Previous               flexFloat64         `json:"previous"`
	LineWeight             flexFloat64         `json:"line_weight"`
	PreviousTimeframePrice IndexSummaryPrice   `json:"previous_timeframe_price"`
	ChartType              string              `json:"chart_type"`
	IntervalInMinutes      int                 `json:"interval_in_minutes"`
	AllowedChartType       []string            `json:"allowed_chart_type"`
	MaxCandles             int                 `json:"max_candles"`
	Prices                 []IndexSummaryPrice `json:"prices"`
}

// IndexSummaryPrice is a single point in the index price series. open/high/low/
// volume are empty strings for the line chart; value/percentage are strings
// while change is a number (int or float depending on the point).
type IndexSummaryPrice struct {
	Date          string      `json:"date"`
	FormattedDate string      `json:"formatted_date"`
	XLabel        string      `json:"xlabel"`
	Value         string      `json:"value"`
	Percentage    string      `json:"percentage"`
	Change        flexFloat64 `json:"change"`
	Open          string      `json:"open"`
	High          string      `json:"high"`
	Low           string      `json:"low"`
	Volume        string      `json:"volume"`
}

// GetIndexSummary returns the index chart summary (price series plus per-day
// summary) for an index symbol such as IHSG. The upstream endpoint only accepts
// the "daily" path segment; granularity is controlled by the interval query
// parameter (e.g. INTERVAL_CHART_MINUTELY). When interval is empty the request
// is sent without it and upstream returns daily points.
func (c *Client) GetIndexSummary(ctx context.Context, symbol, from, to, interval string) (*IndexSummaryResponse, error) {
	q := url.Values{}
	q.Set("from", from)
	q.Set("to", to)
	if interval != "" {
		q.Set("interval", interval)
	}
	var out IndexSummaryResponse
	if err := c.Get(ctx, fmt.Sprintf(indexChartPath, symbol), q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
