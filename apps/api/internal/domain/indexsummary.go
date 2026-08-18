package domain

// IndexSummaryData is the index chart summary (e.g. IHSG): per-day summary
// fields plus the intraday/daily price series.
type IndexSummaryData struct {
	Cagr                   string
	Change                 float64
	Drawdown               string
	MarkingPoint           string
	Percentage             string
	Timeframe              string
	XAxisOpt               string
	Previous               float64
	LineWeight             float64
	PreviousTimeframePrice IndexSummaryPrice
	ChartType              string
	IntervalInMinutes      int
	AllowedChartType       []string
	MaxCandles             int
	Prices                 []IndexSummaryPrice
}

// IndexSummaryPrice is a single point in the index price series.
type IndexSummaryPrice struct {
	Date          string
	FormattedDate string
	XLabel        string
	Value         string
	Percentage    string
	Change        float64
	Open          string
	High          string
	Low           string
	Volume        string
}
