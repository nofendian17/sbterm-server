package stockbit

import (
	"context"
	"net/url"
)

const marketMoverPath = "/order-trade/market-mover"

type MoverType string

const (
	MoverTypeTopGainer      MoverType = "MOVER_TYPE_TOP_GAINER"
	MoverTypeTopLoser       MoverType = "MOVER_TYPE_TOP_LOSER"
	MoverTypeTopValue       MoverType = "MOVER_TYPE_TOP_VALUE"
	MoverTypeTopVolume      MoverType = "MOVER_TYPE_TOP_VOLUME"
	MoverTypeTopFrequency   MoverType = "MOVER_TYPE_TOP_FREQUENCY"
	MoverTypeTopForeignBuy  MoverType = "MOVER_TYPE_NET_FOREIGN_BUY"
	MoverTypeTopForeignSell MoverType = "MOVER_TYPE_NET_FOREIGN_SELL"
	MoverTypeTopIEP         MoverType = "MOVER_TYPE_IEVAL_TOP_GAINER"
)

type FilterStocks string

const (
	FilterStocksMainBoard              FilterStocks = "FILTER_STOCKS_TYPE_MAIN_BOARD"
	FilterStocksDevelopmentBoard       FilterStocks = "FILTER_STOCKS_TYPE_DEVELOPMENT_BOARD"
	FilterStocksAccelerationBoard      FilterStocks = "FILTER_STOCKS_TYPE_ACCELERATION_BOARD"
	FilterStocksNewEconomyBoard        FilterStocks = "FILTER_STOCKS_TYPE_NEW_ECONOMY_BOARD"
	FilterStocksSpecialMonitoringBoard FilterStocks = "FILTER_STOCKS_TYPE_SPECIAL_MONITORING_BOARD"
	FilterStocksWarrantAndRight        FilterStocks = "FILTER_STOCKS_TYPE_WARRANT_AND_RIGHT"
)

type MarketMoverRequest struct {
	MoverType    MoverType
	FilterStocks []FilterStocks
}

// MarketMoverResponse is the market mover response: data.mover_list.
type MarketMoverResponse struct {
	Message string `json:"message"`
	Data    struct {
		MoverList []MarketMover `json:"mover_list"`
	} `json:"data"`
}

type MarketMover struct {
	StockDetail           MoverStockDetail `json:"stock_detail"`
	Price                 float64          `json:"price"`
	Change                MoverChange      `json:"change"`
	Value                 ValueInfo        `json:"value"`
	Volume                ValueInfo        `json:"volume"`
	Frequency             ValueInfo        `json:"frequency"`
	NetForeignBuy         ValueInfo        `json:"net_foreign_buy"`
	NetForeignSell        ValueInfo        `json:"net_foreign_sell"`
	NetBuy                ValueInfo        `json:"net_buy"`
	NetSell               ValueInfo        `json:"net_sell"`
	IEPIEVDetail          IEPIEVDetail     `json:"iepiev_detail"`
	BigMoneyNetValue      ValueInfo        `json:"big_money_net_value"`
	BuyValuePercentage    float64          `json:"buy_value_percentage"`
	SellValuePercentage   float64          `json:"sell_value_percentage"`
	BigMoneyBuyValuePerc  float64          `json:"big_money_buy_value_percentage"`
	BigMoneySellValuePerc float64          `json:"big_money_sell_value_percentage"`
	BidPercent            float64          `json:"bid_percent"`
	CatalogDetail         CatalogDetail    `json:"catalog_detail"`
	MarketCap             any              `json:"market_cap"`
}

type MoverStockDetail struct {
	Code       string          `json:"code"`
	Name       string          `json:"name"`
	IconURL    string          `json:"icon_url"`
	HasUMA     bool            `json:"has_uma"`
	Notations  []Notation      `json:"notations"`
	Corpaction MoverCorpAction `json:"corpaction"`
}

type MoverCorpAction struct {
	Active  bool   `json:"active"`
	IconURL string `json:"icon_url"`
	Text    string `json:"text"`
}

type MoverChange struct {
	Value      float64 `json:"value"`
	Percentage float64 `json:"percentage"`
}

// ValueInfo carries a raw numeric value and its display-formatted string.
type ValueInfo struct {
	Raw       float64 `json:"raw"`
	Formatted string  `json:"formatted"`
}

type IEPIEVDetail struct {
	IEP              ValueInfo `json:"iep"`
	IEV              ValueInfo `json:"iev"`
	IEVAL            ValueInfo `json:"ieval"`
	IEPChange        ValueInfo `json:"iep_change"`
	IEPChangePrev    ValueInfo `json:"iep_change_prev"`
	IEPPriceDiff     ValueInfo `json:"iep_price_diff"`
	IEPPrevPriceDiff ValueInfo `json:"iep_prev_price_diff"`
}

type CatalogDetail struct {
	Code           string `json:"code"`
	Name           string `json:"name"`
	IconURL        string `json:"icon_url"`
	UpCount        int    `json:"up_count"`
	DownCount      int    `json:"down_count"`
	UnchangedCount int    `json:"unchanged_count"`
	StockCount     int    `json:"stock_count"`
	CatalogID      string `json:"catalog_id"`
}

// GetMarketMover returns the market movers (e.g. top gainers) across the
// requested boards. The access token is attached automatically.
func (c *Client) GetMarketMover(ctx context.Context, req MarketMoverRequest) (*MarketMoverResponse, error) {
	query := url.Values{}
	if req.MoverType != "" {
		query.Set("mover_type", string(req.MoverType))
	}
	for _, board := range req.FilterStocks {
		query.Add("filter_stocks", string(board))
	}
	var out MarketMoverResponse
	if err := c.Get(ctx, marketMoverPath, query, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
