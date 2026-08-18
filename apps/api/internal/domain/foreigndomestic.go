package domain

import "encoding/json"

// ForeignDomesticData is the foreign/domestic buy-sell history for a symbol:
// an OHLC price series plus daily foreign-buy/sell aggregates.
type ForeignDomesticData struct {
	HistoricalPrice []ForeignDomesticPricePoint
	HistoricalNet   []ForeignDomesticNetPoint
	LastUpdated     string
	From            string
	To              string
}

// ForeignDomesticPricePoint is one OHLC bar of the price series; raw values are
// display strings upstream.
type ForeignDomesticPricePoint struct {
	Date          string
	DatetimeLabel string
	Open          RawFormatted
	High          RawFormatted
	Low           RawFormatted
	Close         RawFormatted
}

// ForeignDomesticNetPoint is one day's foreign-buy/sell aggregates.
type ForeignDomesticNetPoint struct {
	Date                    string
	DatetimeLabel           string
	DatetimeLabelTable      string
	NetForeign              ForeignDomesticValue
	ForeignBuy              ForeignDomesticValue
	ForeignSell             ForeignDomesticValue
	ForeignFlow             ForeignDomesticValue
	NetLot                  ForeignDomesticValue
	NetFrequency            ForeignDomesticValue
	AveragePrice            ForeignDomesticValue
	PercentageForeignValue  ForeignDomesticValue
	PercentageDomesticValue ForeignDomesticValue
}

// ForeignDomesticValue carries a daily aggregate: upstream raw is a number for
// most fields but a string for net_frequency and a float for the percentages,
// so it stays a json.Number.
type ForeignDomesticValue struct {
	Raw       json.Number
	Formatted string
}
