package stockbit

import (
	"context"
	"fmt"
)

const subsidiaryPath = "/emitten-metadata/subsidiary/%s"

// SubsidiaryResponse is the subsidiary response: data wraps the list.
type SubsidiaryResponse struct {
	Message string         `json:"message"`
	Data    SubsidiaryList `json:"data"`
}

type SubsidiaryList struct {
	Currency          string       `json:"currency"`
	LastUpdatedPeriod string       `json:"last_updated_period"`
	Unit              string       `json:"unit"`
	Subsidiaries      []Subsidiary `json:"subsidiaries"`
}

type Subsidiary struct {
	CompanyName       string  `json:"company_name"`
	BusinessType      string  `json:"business_type"`
	Location          string  `json:"location"`
	CommercialYear    string  `json:"commercial_year"`
	TotalAssets       string  `json:"total_assets"`
	Percentage        string  `json:"percentage"`
	ID                int64   `json:"id"`
	OperationalStatus string  `json:"operational_status"`
	Period            string  `json:"period"`
	Raw               *string `json:"raw"`
}

// GetSubsidiaries returns the subsidiaries of a company. The access token is
// attached automatically.
func (c *Client) GetSubsidiaries(ctx context.Context, symbol string) (*SubsidiaryResponse, error) {
	var out SubsidiaryResponse
	if err := c.Get(ctx, fmt.Sprintf(subsidiaryPath, symbol), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
