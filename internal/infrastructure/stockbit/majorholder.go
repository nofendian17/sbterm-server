package stockbit

import (
	"context"
	"net/url"
	"strconv"
)

const majorHolderPath = "/insider/company/majorholder"

// MajorHolderResponse is the major holder response: data wraps the movement
// list and pagination flag.
type MajorHolderResponse struct {
	Data MajorHolderData `json:"data"`
}

type MajorHolderData struct {
	IsMore   bool                  `json:"is_more"`
	Movement []MajorHolderMovement `json:"movement"`
}

type MajorHolderMovement struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	Symbol         string                 `json:"symbol"`
	Date           string                 `json:"date"`
	Previous       MajorHolderValueChange `json:"previous"`
	Current        MajorHolderValueChange `json:"current"`
	Changes        MajorHolderValueChange `json:"changes"`
	Marker         string                 `json:"marker"`
	IsPosted       bool                   `json:"is_posted"`
	CMHID          string                 `json:"cmh_id"`
	Nationality    string                 `json:"nationality"`
	ActionType     string                 `json:"action_type"`
	DataSource     MajorHolderDataSource  `json:"data_source"`
	PriceFormatted string                 `json:"price_formatted"`
	BrokerDetail   MajorHolderBroker      `json:"broker_detail"`
	Badges         []string               `json:"badges"`
}

type MajorHolderValueChange struct {
	Value          string `json:"value"`
	Percentage     string `json:"percentage"`
	FormattedValue string `json:"formatted_value"`
}

type MajorHolderDataSource struct {
	Label string `json:"label"`
	Type  string `json:"type"`
}

type MajorHolderBroker struct {
	Code  string `json:"code"`
	Group string `json:"group"`
}

// GetMajorHolder returns the major holder movements for a set of symbols. The
// access token is attached automatically.
func (c *Client) GetMajorHolder(ctx context.Context, symbols, actionType, sourceType string, page, limit int) (*MajorHolderResponse, error) {
	q := url.Values{}
	q.Set("symbols", symbols)
	if page > 0 {
		q.Set("page", strconv.Itoa(page))
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	if actionType != "" {
		q.Set("action_type", actionType)
	}
	if sourceType != "" {
		q.Set("source_type", sourceType)
	}
	var out MajorHolderResponse
	if err := c.Get(ctx, majorHolderPath, q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
