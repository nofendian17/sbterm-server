package domain

type PricePerformanceData struct {
	Prices []PricePerformance
}

type PricePerformance struct {
	Close      PriceRawFormatted
	High       PriceRawFormatted
	Low        PriceRawFormatted
	Percentage PricePercent
	Timeframe  string
}

type PriceRawFormatted struct {
	Raw       float64
	Formatted string
}

type PricePercent struct {
	Raw       float64
	Formatted string
}
