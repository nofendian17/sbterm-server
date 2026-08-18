package domain

// IndexChartData combines the index summary (previous close + intraday/daily
// price series) with the chartbit OHLC bars for the same index.
type IndexChartData struct {
	Summary IndexSummaryData
	Chart   ChartPriceData
}
