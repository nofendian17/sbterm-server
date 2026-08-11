package domain

// HistoricalSummaryData is the historical price summary for a symbol: one row
// per period over the requested date range, plus pagination info.
type HistoricalSummaryData struct {
	Result   []HistoricalSummaryItem
	Paginate HistoricalSummaryPaginate
}

// HistoricalSummaryItem is one period row of the summary.
type HistoricalSummaryItem struct {
	Date             string
	Close            float64
	Change           float64
	Value            int64
	Volume           int64
	Frequency        int64
	ForeignBuy       int64
	ForeignSell      int64
	NetForeign       int64
	Open             float64
	High             float64
	Low              float64
	Average          float64
	ChangePercentage float64
}

// HistoricalSummaryPaginate carries the upstream pagination cursor.
type HistoricalSummaryPaginate struct {
	NextPage string
}
