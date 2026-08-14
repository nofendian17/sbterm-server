package domain

import "encoding/json"

// OrderBookData is the order book for a symbol: bid/offer price levels plus
// quote summary and market-wide aggregates.
type OrderBookData struct {
	Average                      int
	Bid                          []OrderBookLevel
	Change                       int
	Close                        int
	Country                      string
	Domestic                     string
	Down                         string
	Exchange                     string
	FBuy                         int64
	FNet                         int64
	Foreign                      string
	Frequency                    int64
	FSell                        int64
	High                         int
	ID                           string
	LastPrice                    int
	Low                          int
	Offer                        []OrderBookLevel
	Open                         int
	PercentageChange             float64
	Previous                     int
	Status                       string
	Symbol                       string
	Symbol2                      string
	Symbol3                      string
	Tradable                     bool
	Unchanged                    string
	Up                           string
	Value                        int64
	Volume                       int64
	CorpAction                   OrderBookCorpAction
	Notation                     []json.RawMessage
	UMA                          bool
	HasForeignBS                 bool
	IEPIEV                       json.RawMessage
	MarketData                   []OrderBookMarket
	Name                         string
	IconURL                      string
	ARA                          OrderBookLimit
	ARB                          OrderBookLimit
	CompanyType                  string
	TotalBidOffer                OrderBookTotal
	NextARA                      OrderBookLimit
	NextARB                      OrderBookLimit
	AutoRejectTimeLeftInSec      int
	AutoRejectEstimation         []json.RawMessage
	OrderbookActiveFeatureMobile string
}

// OrderBookLevel is one price level of the bid or offer side.
type OrderBookLevel struct {
	Price            string
	QueNum           string
	Volume           string
	ChangePercentage string
}

type OrderBookCorpAction struct {
	Active bool
	Icon   string
	Text   string
}

type OrderBookLimit struct {
	Value   string
	Visible bool
}

type OrderBookMarket struct {
	Label     string
	Frequency RawFormatted
	Volume    RawFormatted
	Value     RawFormatted
}

type OrderBookTotal struct {
	Bid        OrderBookSide
	Offer      OrderBookSide
	BidPercent float64
}

type OrderBookSide struct {
	Freq    string
	Lot     string
	RawLot  string
	RawFreq string
}
