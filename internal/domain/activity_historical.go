package domain

// ActivityHistoricalData is the broker activity historical response: per-interval
// trade/price records for the requested symbols and broker codes over a date
// range, plus a net-value summary grouped by the requested net interval.
type ActivityHistoricalData struct {
	DateFrom    string
	DateTo      string
	Symbols     []string
	BrokerCodes []string
	BrokerName  string
	Records     []ActivityHistoricalRecord
	Pagination  ActivityHistoricalPaginate
	Summary     ActivityHistoricalSummary
}

// ActivityHistoricalRecord is one interval bucket: the aggregated trade and
// price activity for a date (and optionally a single broker).
type ActivityHistoricalRecord struct {
	Date          string
	BrokerCode    string
	TradeActivity ActivityHistoricalTrade
	PriceActivity ActivityHistoricalPrice
}

// ActivityHistoricalTrade aggregates buy/sell/net values and lot shares.
type ActivityHistoricalTrade struct {
	NetSummary     ActivitySummary
	BuySummary     ActivitySummary
	SellSummary    ActivitySummary
	ForeignSummary ActivityForeignSummary
	TotalBuyLot    ActivityLotShare
	TotalSellLot   ActivityLotShare
}

// ActivitySummary is one value/volume/frequency summary block.
type ActivitySummary struct {
	AveragePrice float64
	Frequency    float64
	Lot          float64
	Value        float64
}

// ActivityForeignSummary is the foreign buy/sell flow summary.
type ActivityForeignSummary struct {
	ForeignBuy  float64
	ForeignSell float64
	NetForeign  float64
}

// ActivityLotShare is one lot side's absolute amount and market share percent.
type ActivityLotShare struct {
	Amount float64
	Pct    float64
}

// ActivityHistoricalPrice is the price move for the interval bucket. ClosePrice
// is a string, matching the upstream's string serialization.
type ActivityHistoricalPrice struct {
	ClosePrice    string
	ReturnSummary ActivityHistoricalPriceReturn
}

// ActivityHistoricalPriceReturn is the absolute and percent price change.
type ActivityHistoricalPriceReturn struct {
	Amount float64
	Pct    float64
}

// ActivityHistoricalPaginate is the paging info for the records list.
type ActivityHistoricalPaginate struct {
	Page    int
	Limit   int
	HasNext bool
	HasPrev bool
}

// ActivityHistoricalSummary is the net-value summary grouped by the requested
// net interval (e.g. monthly buckets over the whole range).
type ActivityHistoricalSummary struct {
	GroupType string
	Data      []ActivityHistoricalSummaryGroup
}

// ActivityHistoricalSummaryGroup is one net-interval bucket of the summary.
type ActivityHistoricalSummaryGroup struct {
	DateFrom   string
	DateTo     string
	NetSummary ActivitySummary
}
