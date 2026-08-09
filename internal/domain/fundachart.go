package domain

type FundaChartCompany struct {
	CompanyID   int64
	CompanyName string
	Ratios      []FundaChartRatio
}

type FundaChartRatio struct {
	DecimalPoint int
	GroupData    bool
	ItemID       int64
	ItemName     string
	ItemType     int
	Suffix       string
	XAxisID      int
	YAxisID      int
	ChartData    []FundaChartPoint
}

type FundaChartPoint struct {
	Date         int64
	FormatedDate string
	Value        float64
	RatioValue   float64
}
