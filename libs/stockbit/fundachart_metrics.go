package stockbit

import (
	"context"
	"net/url"
)

const fundaChartMetricsPath = "/fundachart/metrics"

// FundaChartMetricsResponse is the metrics catalog response: data is the
// recursive metric tree.
type FundaChartMetricsResponse struct {
	Message string             `json:"message"`
	Data    []FundaChartMetric `json:"data"`
}

type FundaChartMetric struct {
	FitemID       int64              `json:"fitem_id"`
	FitemName     string             `json:"fitem_name"`
	ShowChartIcon int                `json:"show_chart_icon"`
	Child         []FundaChartMetric `json:"child"`
}

// GetFundaChartMetrics returns the available fundachart metrics. The access
// token is attached automatically.
func (c *Client) GetFundaChartMetrics(ctx context.Context, metricName string) (*FundaChartMetricsResponse, error) {
	q := url.Values{}
	q.Set("metric_name", metricName)
	var out FundaChartMetricsResponse
	if err := c.Get(ctx, fundaChartMetricsPath, q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
