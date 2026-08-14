package stockbit

import (
	"context"
	"encoding/json"
	"fmt"
)

const orderBookPath = "/company-price-feed/v2/orderbook/companies/%s"

// OrderBookResponse is the orderbook response for a symbol.
type OrderBookResponse struct {
	Data OrderBookData `json:"data"`
}

type OrderBookData struct {
	Average                      int                 `json:"average"`
	Bid                          []OrderBookLevel    `json:"bid"`
	Change                       int                 `json:"change"`
	Close                        int                 `json:"close"`
	Country                      string              `json:"country"`
	Domestic                     string              `json:"domestic"`
	Down                         string              `json:"down"`
	Exchange                     string              `json:"exchange"`
	FBuy                         int64               `json:"fbuy"`
	FNet                         int64               `json:"fnet"`
	Foreign                      string              `json:"foreign"`
	Frequency                    int64               `json:"frequency"`
	FSell                        int64               `json:"fsell"`
	High                         int                 `json:"high"`
	ID                           string              `json:"id"`
	LastPrice                    int                 `json:"lastprice"`
	Low                          int                 `json:"low"`
	Offer                        []OrderBookLevel    `json:"offer"`
	Open                         int                 `json:"open"`
	PercentageChange             float64             `json:"percentage_change"`
	Previous                     int                 `json:"previous"`
	Status                       string              `json:"status"`
	Symbol                       string              `json:"symbol"`
	Symbol2                      string              `json:"symbol_2"`
	Symbol3                      string              `json:"symbol_3"`
	Tradable                     bool                `json:"tradable"`
	Unchanged                    string              `json:"unchanged"`
	Up                           string              `json:"up"`
	Value                        int64               `json:"value"`
	Volume                       int64               `json:"volume"`
	CorpAction                   OrderBookCorpAction `json:"corp_action"`
	Notation                     []json.RawMessage   `json:"notation"`
	UMA                          bool                `json:"uma"`
	HasForeignBS                 bool                `json:"has_foreign_bs"`
	IEPIEV                       json.RawMessage     `json:"iepiev"`
	MarketData                   []OrderBookMarket   `json:"market_data"`
	Name                         string              `json:"name"`
	IconURL                      string              `json:"icon_url"`
	ARA                          OrderBookLimit      `json:"ara"`
	ARB                          OrderBookLimit      `json:"arb"`
	CompanyType                  string              `json:"company_type"`
	TotalBidOffer                OrderBookTotal      `json:"total_bid_offer"`
	NextARA                      OrderBookLimit      `json:"next_ara"`
	NextARB                      OrderBookLimit      `json:"next_arb"`
	AutoRejectTimeLeftInSec      int                 `json:"autoreject_time_left_in_sec"`
	AutoRejectEstimation         []json.RawMessage   `json:"auto_reject_estimation"`
	OrderbookActiveFeatureMobile string              `json:"orderbook_active_feature_mobile"`
}

// OrderBookLevel is one price level of the bid or offer side.
type OrderBookLevel struct {
	Price            string `json:"price"`
	QueNum           string `json:"que_num"`
	Volume           string `json:"volume"`
	ChangePercentage string `json:"change_percentage"`
}

type OrderBookCorpAction struct {
	Active bool   `json:"active"`
	Icon   string `json:"icon"`
	Text   string `json:"text"`
}

type OrderBookLimit struct {
	Value   string `json:"value"`
	Visible bool   `json:"visible"`
}

type OrderBookMarket struct {
	Label     string                `json:"label"`
	Frequency OrderBookRawFormatted `json:"frequency"`
	Volume    OrderBookRawFormatted `json:"volume"`
	Value     OrderBookRawFormatted `json:"value"`
}

type OrderBookRawFormatted struct {
	Raw       string `json:"raw"`
	Formatted string `json:"formatted"`
}

type OrderBookTotal struct {
	Bid        OrderBookSide `json:"bid"`
	Offer      OrderBookSide `json:"offer"`
	BidPercent float64       `json:"bid_percent"`
}

type OrderBookSide struct {
	Freq    string `json:"freq"`
	Lot     string `json:"lot"`
	RawLot  string `json:"raw_lot"`
	RawFreq string `json:"raw_freq"`
}

// GetOrderBook returns the order book for a symbol. The access token is
// attached automatically; the endpoint works with any X-Platform header.
func (c *Client) GetOrderBook(ctx context.Context, symbol string) (*OrderBookResponse, error) {
	var out OrderBookResponse
	if err := c.Get(ctx, fmt.Sprintf(orderBookPath, symbol), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
