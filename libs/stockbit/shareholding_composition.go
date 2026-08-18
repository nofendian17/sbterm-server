package stockbit

import (
	"context"
	"fmt"
	"net/url"
)

const shareholdingCompositionPath = "/insider/shareholding/composition/companies/%s"

// ShareholdingCompositionResponse is the shareholding composition response:
// data wraps the periods list.
type ShareholdingCompositionResponse struct {
	Data ShareholdingCompositionData `json:"data"`
}

type ShareholdingCompositionData struct {
	Periods []ShareholdingCompositionPeriod `json:"periods"`
}

type ShareholdingCompositionPeriod struct {
	ReportDate   string                    `json:"report_date"`
	TotalShares  ShareholdingRawFormatted  `json:"total_shares"`
	Compositions []ShareholdingComposition `json:"compositions"`
}

type ShareholdingComposition struct {
	Label      string                   `json:"label"`
	Shares     ShareholdingRawFormatted `json:"shares"`
	Percentage ShareholdingPercent      `json:"percentage"`
	Colors     ShareholdingColors       `json:"colors"`
}

type ShareholdingRawFormatted struct {
	Raw       string `json:"raw"`
	Formatted string `json:"formatted"`
}

type ShareholdingPercent struct {
	Raw       float64 `json:"raw"`
	Formatted string  `json:"formatted"`
}

type ShareholdingColors struct {
	Light string `json:"light"`
	Dark  string `json:"dark"`
}

// GetShareholdingComposition returns the insider shareholding composition for
// a company, optionally filtered by report period. The access token is
// attached automatically.
func (c *Client) GetShareholdingComposition(ctx context.Context, symbol, periodStart, periodEnd string) (*ShareholdingCompositionResponse, error) {
	q := url.Values{}
	if periodStart != "" {
		q.Set("period_start", periodStart)
	}
	if periodEnd != "" {
		q.Set("period_end", periodEnd)
	}
	var out ShareholdingCompositionResponse
	if err := c.Get(ctx, fmt.Sprintf(shareholdingCompositionPath, symbol), q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
