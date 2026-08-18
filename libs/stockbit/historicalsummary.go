package stockbit

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

const historicalSummaryPath = "/company-price-feed/historical/summary/%s"

// HistoricalSummaryResponse is the historical price summary response.
type HistoricalSummaryResponse struct {
	Message string                `json:"message"`
	Data    HistoricalSummaryData `json:"data"`
}

type HistoricalSummaryData struct {
	Result   []HistoricalSummaryItem   `json:"result"`
	Paginate HistoricalSummaryPaginate `json:"paginate"`
}

// HistoricalSummaryItem is one period row of the summary (close/open/high/low,
// change, value/volume, and foreign flow).
type HistoricalSummaryItem struct {
	Date             string  `json:"date"`
	Close            float64 `json:"close"`
	Change           float64 `json:"change"`
	Value            int64   `json:"value"`
	Volume           int64   `json:"volume"`
	Frequency        int64   `json:"frequency"`
	ForeignBuy       int64   `json:"foreign_buy"`
	ForeignSell      int64   `json:"foreign_sell"`
	NetForeign       int64   `json:"net_foreign"`
	Open             float64 `json:"open"`
	High             float64 `json:"high"`
	Low              float64 `json:"low"`
	Average          float64 `json:"average"`
	ChangePercentage float64 `json:"change_percentage"`
}

type HistoricalSummaryPaginate struct {
	NextPage string `json:"next_page"`
}

// GetHistoricalSummary returns the historical price summary for a symbol over a
// date range, aggregated by the given period (HS_PERIOD_*). limit/page control
// pagination; the access token is attached automatically.
func (c *Client) GetHistoricalSummary(ctx context.Context, symbol, period, startDate, endDate string, limit, page int) (*HistoricalSummaryResponse, error) {
	q := url.Values{}
	q.Set("period", period)
	q.Set("start_date", startDate)
	q.Set("end_date", endDate)
	q.Set("limit", strconv.Itoa(limit))
	q.Set("page", strconv.Itoa(page))
	var out HistoricalSummaryResponse
	if err := c.Get(ctx, fmt.Sprintf(historicalSummaryPath, symbol), q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
