package domain

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
