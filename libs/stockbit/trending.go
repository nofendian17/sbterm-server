package stockbit

import "context"

const trendingPath = "/emitten/trending"

// TrendingResponse is the trending stocks response: data is the list itself.
type TrendingResponse struct {
	Message string          `json:"message"`
	Data    []TrendingStock `json:"data"`
}

type TrendingStock struct {
	Change           string           `json:"change"`
	Symbol           string           `json:"symbol"`
	Percent          string           `json:"percent"`
	Name             string           `json:"name"`
	Last             string           `json:"last"`
	Symbol2          string           `json:"symbol_2"`
	Symbol3          string           `json:"symbol_3"`
	CompanyID        string           `json:"company_id"`
	Notation         []Notation       `json:"notation"`
	UMA              bool             `json:"uma"`
	Tradeable        int              `json:"tradeable"`
	Country          string           `json:"country"`
	Type             string           `json:"type"`
	CorpAction       CorpAction       `json:"corp_action"`
	IsExist          int              `json:"isexist"`
	Status           string           `json:"status"`
	IconURL          string           `json:"icon_url"`
	IsFollowing      bool             `json:"is_following"`
	FormattedPrice   string           `json:"formatted_price"`
	IsExists         bool             `json:"is_exists"`
	Previous         string           `json:"previous"`
	DayTradeInfo     DayTradeInfo     `json:"day_trade_info"`
	TradingLimitInfo TradingLimitInfo `json:"trading_limit_info"`
	MarginInfo       MarginInfo       `json:"margin_info"`
}

type CorpAction struct {
	Active bool   `json:"active"`
	Icon   string `json:"icon"`
	Text   string `json:"text"`
	Detail any    `json:"detail"`
}

// Notation is a stock exchange special notation (e.g. OJK sanctions).
type Notation struct {
	NotationCode string  `json:"notation_code"`
	NotationDesc string  `json:"notation_desc"`
	IconURL      IconURL `json:"icon_url"`
}

type IconURL struct {
	LightMode string `json:"light_mode"`
	DarkMode  string `json:"dark_mode"`
}

type DayTradeInfo struct {
	IsShowMultiplier bool   `json:"is_show_multiplier"`
	Multiplier       string `json:"multiplier"`
}

type TradingLimitInfo struct {
	IsTradingLimit    bool   `json:"is_trading_limit"`
	HaircutPercentage string `json:"haircut_percentage"`
}

type MarginInfo struct {
	IsMarginTrading bool    `json:"is_margin_trading"`
	Percentage      string  `json:"percentage"`
	PercentageRaw   float64 `json:"percentage_raw"`
}

// GetTrending returns the currently trending stocks. The access token is
// attached automatically.
func (c *Client) GetTrending(ctx context.Context) (*TrendingResponse, error) {
	var out TrendingResponse
	if err := c.Get(ctx, trendingPath, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
