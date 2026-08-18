package stockbit

import (
	"context"
	"net/url"
)

const fundaChartPath = "/fundachart"

// FundaChartResponse is the fundachart response: data is the per-company
// ratio series list.
type FundaChartResponse struct {
	Message string              `json:"message"`
	Data    []FundaChartCompany `json:"data"`
}

type FundaChartCompany struct {
	CompanyID   int64             `json:"company_id"`
	CompanyName string            `json:"company_name"`
	Ratios      []FundaChartRatio `json:"ratios"`
}

type FundaChartRatio struct {
	DecimalPoint int               `json:"decimal_point"`
	GroupData    bool              `json:"group_data"`
	ItemID       int64             `json:"item_id"`
	ItemName     string            `json:"item_name"`
	ItemType     int               `json:"item_type"`
	Suffix       string            `json:"suffix"`
	XAxisID      int               `json:"xaxis_id"`
	YAxisID      int               `json:"yaxis_id"`
	ChartData    []FundaChartPoint `json:"chart_data"`
}

type FundaChartPoint struct {
	Date         int64   `json:"date"`
	FormatedDate string  `json:"formated_date"`
	Value        float64 `json:"value"`
	RatioValue   float64 `json:"ratio_value"`
}

// GetFundaChart returns the raw historical ratio series for one or more
// fin-items (comma-separated) of a company. The access token is attached
// automatically.
func (c *Client) GetFundaChart(ctx context.Context, symbol, item, timeframe string) (*FundaChartResponse, error) {
	q := url.Values{}
	q.Set("companies", symbol)
	q.Set("item", item)
	q.Set("timeframe", timeframe)
	var out FundaChartResponse
	if err := c.Get(ctx, fundaChartPath, q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
