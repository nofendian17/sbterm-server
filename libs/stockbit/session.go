package stockbit

import "context"

const marketSessionPath = "/company-price-feed/market-time/session"

// MarketSessionResponse is the market session response: data.detail.
type MarketSessionResponse struct {
	Message string `json:"message"`
	Data    struct {
		Datetime string              `json:"datetime"`
		Detail   MarketSessionDetail `json:"detail"`
	} `json:"data"`
}

type MarketSessionDetail struct {
	FCA     SessionInfo `json:"fca"`
	Regular SessionInfo `json:"regular"`
}

// SessionInfo describes the trading session state for one market segment.
type SessionInfo struct {
	Session        int       `json:"session"`
	StateName      string    `json:"state_name"`
	IsLastSession  bool      `json:"is_last_session"`
	IsEndOfDay     bool      `json:"is_end_of_day"`
	StateStartTime string    `json:"state_start_time"`
	StateEndTime   string    `json:"state_end_time"`
	TimeLeft       ValueInfo `json:"time_left"`
	SuspendInfo    string    `json:"suspend_info"`
}

// GetMarketSession returns the current market session state. The access token
// is attached automatically.
func (c *Client) GetMarketSession(ctx context.Context) (*MarketSessionResponse, error) {
	var out MarketSessionResponse
	if err := c.Get(ctx, marketSessionPath, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
