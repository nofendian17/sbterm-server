package domain

type FundaChartMetric struct {
	FitemID       int64
	FitemName     string
	ShowChartIcon int
	Child         []FundaChartMetric
}
