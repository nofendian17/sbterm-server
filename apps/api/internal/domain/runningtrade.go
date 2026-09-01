package domain

import "encoding/json"

// RunningTradeData is the running trade chart for a symbol: the price series
// plus per-broker value/volume series for the requested range or period.
type RunningTradeData struct {
	From            string
	To              string
	DataLastUpdated string
	PriceChartData  []RunningTradePricePoint
	BrokerChartData []RunningTradeBrokerGroup
	DateSessionInfo string
}

// RunningTradePricePoint is a single point of the price series. open/high/low
// are pointers because upstream omits them (null) on some series.
type RunningTradePricePoint struct {
	Date          string
	Time          string
	Value         RawFormatted
	DatetimeLabel string
	Open          *RawFormatted
	High          *RawFormatted
	Low           *RawFormatted
}

// RunningTradeBrokerGroup is one broker chart series (value or volume) covering
// the requested broker codes.
type RunningTradeBrokerGroup struct {
	Type    string
	Brokers []string
	Charts  []RunningTradeBrokerChart
}

// RunningTradeBrokerChart is a single broker's series within a group.
type RunningTradeBrokerChart struct {
	BrokerCode string
	Chart      []RunningTradeChartPoint
}

// RunningTradeChartPoint is a single point of a broker series.
type RunningTradeChartPoint struct {
	Date          string
	Time          string
	Value         RawFormatted
	DatetimeLabel string
	Open          *RawFormatted
	High          *RawFormatted
	Low           *RawFormatted
}

// RunningTradeFeed is the running trade feed for a single symbol.
type RunningTradeFeed struct {
	IsOpenMarket bool
	RunningTrade []RunningTradeFeedItem
}

// RunningTradeFeedItem is one executed trade in the feed.
type RunningTradeFeedItem struct {
	ID               string
	Time             string
	Action           string
	Code             string
	Price            string
	Change           string
	Lot              string
	IsBrokerExists   bool
	Buyer            string
	Seller           string
	TradeNumber      string
	BuyerType        string
	SellerType       string
	MarketBoard      string
	BuyOrderNumber   string
	SellOrderNumber  string
	GroupOrderNumber string
	Value            RunningTradeFeedValue
}

// RunningTradeFeedValue carries a trade's value: raw is numeric upstream.
type RunningTradeFeedValue struct {
	Raw       json.Number
	Formatted string
}

// RunningTradeGroupFeed is the grouped running trade feed for a single symbol.
type RunningTradeGroupFeed struct {
	Total             RunningTradeGroupTotal
	RunningTradeGroup []RunningTradeGroupItem
	Date              string
	SingleOrder       bool
}

// RunningTradeGroupTotal holds aggregated totals for the group feed.
type RunningTradeGroupTotal struct {
	Value     RawFormatted
	Lot       RawFormatted
	Frequency RawFormatted
}

// RunningTradeGroupItem is one grouped trade in the feed.
type RunningTradeGroupItem struct {
	ID             string
	OrderNumber    string
	Action         string
	GroupAction    string
	Time           string
	TradeNumber    string
	Code           string
	MarketBoard    string
	Price          RawFormatted
	Change         RawFormatted
	Lot            RawFormatted
	Freq           RawFormatted
	IsBrokerExists bool
	Buyer          []RunningTradeGroupBroker
	Seller         []RunningTradeGroupBroker
	Value          RawFormatted
}

// RunningTradeGroupBroker is a broker entry in a group item's buyer/seller list.
type RunningTradeGroupBroker struct {
	BrokerCode string
	BrokerType string
}
