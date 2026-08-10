package stockbit

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

const marketDetectorPath = "/marketdetectors/%s"

// MarketDetectorResponse is the market detector response. data wraps the
// bandar/detector aggregates and broker summaries.
type MarketDetectorResponse struct {
	Data MarketDetectorData `json:"data"`
}

type MarketDetectorData struct {
	BandarDetector BandarDetector `json:"bandar_detector"`
	BrokerSummary  BrokerSummary  `json:"broker_summary"`
	From           string         `json:"from"`
	To             string         `json:"to"`
}

type BandarDetector struct {
	Average             float64       `json:"average"`
	Avg                 BandarAccdist `json:"avg"`
	Avg5                BandarAccdist `json:"avg5"`
	BrokerAccdist       string        `json:"broker_accdist"`
	NumberBrokerBuysell int           `json:"number_broker_buysell"`
	Top1                BandarAccdist `json:"top1"`
	Top3                BandarAccdist `json:"top3"`
	Top5                BandarAccdist `json:"top5"`
	Top10               BandarAccdist `json:"top10"`
	TotalBuyer          int           `json:"total_buyer"`
	TotalSeller         int           `json:"total_seller"`
	Value               int64         `json:"value"`
	Volume              int64         `json:"volume"`
}

type BandarAccdist struct {
	Accdist string  `json:"accdist"`
	Amount  int64   `json:"amount"`
	Percent float64 `json:"percent"`
	Vol     float64 `json:"vol"`
}

type BrokerSummary struct {
	BrokersBuy  []BrokerBuy  `json:"brokers_buy"`
	BrokersSell []BrokerSell `json:"brokers_sell"`
	Symbol      string       `json:"symbol"`
}

type BrokerBuy struct {
	Blot             string `json:"blot"`
	Blotv            string `json:"blotv"`
	Bval             string `json:"bval"`
	Bvalv            string `json:"bvalv"`
	NetbsBrokerCode  string `json:"netbs_broker_code"`
	NetbsBuyAvgPrice string `json:"netbs_buy_avg_price"`
	NetbsDate        string `json:"netbs_date"`
	NetbsStockCode   string `json:"netbs_stock_code"`
	Type             string `json:"type"`
	Freq             string `json:"freq"`
}

type BrokerSell struct {
	NetbsBrokerCode   string `json:"netbs_broker_code"`
	NetbsDate         string `json:"netbs_date"`
	NetbsSellAvgPrice string `json:"netbs_sell_avg_price"`
	NetbsStockCode    string `json:"netbs_stock_code"`
	Slot              string `json:"slot"`
	Slotv             string `json:"slotv"`
	Sval              string `json:"sval"`
	Svalv             string `json:"svalv"`
	Type              string `json:"type"`
	Freq              string `json:"freq"`
}

// GetMarketDetector returns bandar detector aggregates and broker summaries for
// a symbol. from must be earlier than to. transactionType, marketBoard and
// investorType are the TRANSACTION_TYPE_*, MARKET_BOARD_* and INVESTOR_TYPE_*
// enum values; the access token is attached automatically.
func (c *Client) GetMarketDetector(ctx context.Context, symbol, from, to, transactionType, marketBoard, investorType string, limit int) (*MarketDetectorResponse, error) {
	q := url.Values{}
	q.Set("from", from)
	q.Set("to", to)
	q.Set("transaction_type", transactionType)
	q.Set("market_board", marketBoard)
	q.Set("investor_type", investorType)
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	var out MarketDetectorResponse
	if err := c.Get(ctx, fmt.Sprintf(marketDetectorPath, symbol), q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}