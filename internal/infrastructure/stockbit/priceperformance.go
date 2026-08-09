package stockbit

import (
	"context"
	"fmt"
)

const pricePerformancePath = "/company-price-feed/price-performance/%s"

// PricePerformanceResponse is the price performance response: data wraps the
// per-timeframe price list.
type PricePerformanceResponse struct {
	Message string               `json:"message"`
	Data    PricePerformanceData `json:"data"`
}

type PricePerformanceData struct {
	Prices []PricePerformance `json:"prices"`
}

type PricePerformance struct {
	Close      PriceRawFormatted `json:"close"`
	High       PriceRawFormatted `json:"high"`
	Low        PriceRawFormatted `json:"low"`
	Percentage PricePercent      `json:"percentage"`
	Timeframe  string            `json:"timeframe"`
}

type PriceRawFormatted struct {
	Raw       float64 `json:"raw"`
	Formatted string  `json:"formatted"`
}

type PricePercent struct {
	Raw       float64 `json:"raw"`
	Formatted string  `json:"formatted"`
}

// GetPricePerformance returns the price performance across timeframes for a
// symbol. The access token is attached automatically.
func (c *Client) GetPricePerformance(ctx context.Context, symbol string) (*PricePerformanceResponse, error) {
	var out PricePerformanceResponse
	if err := c.Get(ctx, fmt.Sprintf(pricePerformancePath, symbol), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
