package domain

type ChartPriceData struct {
	AllowDecimal int
	Chartbit     []ChartPrice
}

type ChartPrice struct {
	Date             string
	Unixdate         int64
	Datetime         string
	UnixTimestamp    string
	Open             float64
	High             float64
	Low              float64
	Close            float64
	Volume           float64
	Value            float64
	Frequency        float64
	ForeignBuy       float64
	ForeignSell      float64
	ForeignFlow      float64
	SoxClose         float64
	Dividend         float64
	ShareOutstanding float64
	FreqAnalyzer     float64
	Lot              float64
	ForeignBuyToday  float64
	ForeignSellToday float64
	Symbol           string
}
