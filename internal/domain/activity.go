package domain

// ActivityChartData is the broker activity chart response for the requested
// broker codes over a date range, grouped by chart type
// (TYPE_CHART_VALUE / TYPE_CHART_VOLUME).
type ActivityChartData struct {
	From            string
	To              string
	DataLastUpdated string
	ChartData       []ActivityChartGroup
	DateSessionInfo string
	BrokerCode      []string
	BrokerName      string
}

// ActivityChartGroup is one chart series group for a set of symbols.
type ActivityChartGroup struct {
	Type    string
	Symbols []string
	Charts  []ActivityChartSeries
}

// ActivityChartSeries is a single symbol's series within a chart group.
type ActivityChartSeries struct {
	Symbol string
	Chart  []ActivityChartPoint
}

// ActivityChartPoint is a single point of a symbol's series. Value is a
// raw/formatted pair because upstream returns both representations.
type ActivityChartPoint struct {
	Date          string
	Time          string
	Value         RawFormatted
	DatetimeLabel string
}

// ActivityData is the broker activity transaction response: flat buy/sell
// trading rows for the requested broker codes over a date range.
type ActivityData struct {
	BrokerActivityTransaction BrokerActivityTransaction
	From                      string
	To                        string
	BrokerCode                string
	BrokerName                string
}

// BrokerActivityTransaction holds the buy and sell activity row lists.
type BrokerActivityTransaction struct {
	BrokersBuy  []BrokerActivity
	BrokersSell []BrokerActivity
}

// BrokerActivity is one flat trading row (a broker's trade on a stock on a
// date). Value/lot/freq are numbers; lot and avg_price can be fractional.
type BrokerActivity struct {
	StockCode     string
	BrokerCode    string
	Type          string
	Date          string
	Value         float64
	Lot           float64
	AveragePrice  float64
	Frequency     float64
	CompanyDetail ActivityCompanyDetail
	NetValueTrend []ActivityNetValueTrend
}

// ActivityCompanyDetail holds display metadata for the traded stock.
type ActivityCompanyDetail struct {
	IconURL    string
	CorpAction ActivityCorpAction
	Notation   []ActivityNotation
}

// ActivityNotation is one exchange/regulator notation flag on the stock.
type ActivityNotation struct {
	NotationCode string
	NotationDesc string
	IconURL      ActivityNotationIcon
}

type ActivityNotationIcon struct {
	LightMode string
	DarkMode  string
}

// ActivityCorpAction is the corp-action flag object on a traded stock.
type ActivityCorpAction struct {
	Active bool
	Icon   string
	Text   string
}

// ActivityNetValueTrend is one point of the net-value trend for a trading row.
type ActivityNetValueTrend struct {
	Date  string
	NVal  float64
	NVol  float64
	NFreq float64
}
