package stockbit

import (
	"context"
	"fmt"
)

const subsectorCompaniesPath = "/emitten/v3/sector/%s/subsector/%s/company"

// SubsectorCompaniesResponse is the subsector companies response: data is the
// company list itself.
type SubsectorCompaniesResponse struct {
	Message string             `json:"message"`
	Data    []SubsectorCompany `json:"data"`
}

type SubsectorCompany struct {
	AvgVolume     string `json:"avgvolume"`
	Change        string `json:"change"`
	CompanyID     string `json:"company_id"`
	CompanyStatus string `json:"company_status"`
	Last          string `json:"last"`
	MarketCap     string `json:"marketcap"`
	Name          string `json:"name"`
	Symbol        string `json:"symbol"`
	Value         int64  `json:"value"`
	Volume        int64  `json:"volume"`
	IconURL       string `json:"icon_url"`
	Percent       string `json:"percent"`
	UMA           bool   `json:"uma"`
}

// GetSubsectorCompanies returns the companies in a subsector. The access token
// is attached automatically.
func (c *Client) GetSubsectorCompanies(ctx context.Context, sectorID, subsectorID string) (*SubsectorCompaniesResponse, error) {
	var out SubsectorCompaniesResponse
	if err := c.Get(ctx, fmt.Sprintf(subsectorCompaniesPath, sectorID, subsectorID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
