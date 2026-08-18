package stockbit

import (
	"context"
	"net/url"
	"strconv"
)

const topStockPath = "/order-trade/top-stock"

// TopStockResponse is the top-stock response. data wraps the buy/sell leaderboards
// and paging/display metadata.
type TopStockResponse struct {
	Data TopStockData `json:"data"`
}

type TopStockData struct {
	TopBuy        []TopStockItem        `json:"top_buy"`
	TopSell       []TopStockItem        `json:"top_sell"`
	Total         []TopStockItem        `json:"total"`
	ResponseInfo  TopStockResponseInfo  `json:"response_info"`
	DisplayOption TopStockDisplayOption `json:"display_option"`
}

type TopStockItem struct {
	Rank         int           `json:"rank"`
	Code         string        `json:"code"`
	IconURL      string        `json:"icon_url"`
	Value        TopStockValue `json:"value"`
	Lot          TopStockValue `json:"lot"`
	Average      TopStockValue `json:"average"`
	ForeignValue TopStockValue `json:"foreign_value"`
	Frequency    TopStockValue `json:"frequency"`
}

type TopStockValue struct {
	Raw       string `json:"raw"`
	Formatted string `json:"formatted"`
}

type TopStockResponseInfo struct {
	Page           int    `json:"page"`
	Limit          int    `json:"limit"`
	MaxDayDuration int    `json:"max_day_duration"`
	StartDate      string `json:"start_date"`
	EndDate        string `json:"end_date"`
	ValueType      string `json:"value_type"`
}

type TopStockDisplayOption struct {
	BannerMessage      string                   `json:"banner_message"`
	ForeignValueColumn bool                     `json:"foreign_value_column"`
	EnabledValueType   TopStockEnabledValueType `json:"enabled_value_type"`
}

type TopStockEnabledValueType struct {
	Gross bool `json:"gross"`
	Net   bool `json:"net"`
	Total bool `json:"total"`
}

// GetTopStock returns the top buy/sell leaderboards for a symbol range. start must
// be earlier than end. investorType, marketType and valueType take the
// INVESTOR_TYPE_*, MARKET_TYPE_* and VALUE_TYPE_* enum values; the access token
// is attached automatically.
func (c *Client) GetTopStock(ctx context.Context, start, end, investorType, marketType, valueType string, page int) (*TopStockResponse, error) {
	q := url.Values{}
	q.Set("start", start)
	q.Set("end", end)
	q.Set("investor_type", investorType)
	q.Set("market_type", marketType)
	q.Set("value_type", valueType)
	if page > 0 {
		q.Set("page", strconv.Itoa(page))
	}
	var out TopStockResponse
	if err := c.Get(ctx, topStockPath, q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
