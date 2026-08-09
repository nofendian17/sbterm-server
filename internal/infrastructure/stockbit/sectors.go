package stockbit

import (
	"context"
	"net/url"
)

const sectorsPath = "/emitten/company/catalog"

// SectorsRequest carries the optional sectors query parameters.
type SectorsRequest struct {
	SetPrice string
	SortBy   string
}

// SectorsResponse is the company sectors response: data.pchange_info.
type SectorsResponse struct {
	Message string `json:"message"`
	Data    struct {
		PChangeInfo []Sector `json:"pchange_info"`
	} `json:"data"`
}

type Sector struct {
	Icon     string   `json:"icon"`
	Prices   []string `json:"prices"`
	Previous float64  `json:"previous"`
	Last     float64  `json:"last"`
	Change   string   `json:"change"`
	Percent  float64  `json:"percent"`
	Type     string   `json:"type"`
	Symbol   string   `json:"symbol"`
	Symbol2  string   `json:"symbol_2"`
	ID       string   `json:"id"`
}

// GetSectors returns the company sectors. The access token is attached
// automatically.
func (c *Client) GetSectors(ctx context.Context, req SectorsRequest) (*SectorsResponse, error) {
	query := url.Values{}
	if req.SetPrice != "" {
		query.Set("setprice", req.SetPrice)
	}
	if req.SortBy != "" {
		query.Set("sortby", req.SortBy)
	}
	var out SectorsResponse
	if err := c.Get(ctx, sectorsPath, query, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
