package stockbit

import "context"

const indexesPath = "/emitten/indexes/mobile"

// IndexResponse is the market indexes response: data.{main,all}.
type IndexResponse struct {
	Message string `json:"message"`
	Data    struct {
		Main []Index `json:"main"`
		All  []Index `json:"all"`
	} `json:"data"`
}

type Index struct {
	Parent    int64  `json:"parent"`
	ID        string `json:"id"`
	Symbol    string `json:"symbol"`
	Name      string `json:"name"`
	Percent   string `json:"percent"`
	Change    string `json:"change"`
	Last      string `json:"last"`
	MarketCap string `json:"marketcap"`
	ValueMA20 string `json:"valuema20"`
}

// GetIndexes returns the market indexes (main and all lists). The access token
// is attached automatically.
func (c *Client) GetIndexes(ctx context.Context) (*IndexResponse, error) {
	var out IndexResponse
	if err := c.Get(ctx, indexesPath, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
