package domain

type ShareholdingCompositionPeriod struct {
	ReportDate   string
	TotalShares  ShareholdingRawFormatted
	Compositions []ShareholdingComposition
}

type ShareholdingComposition struct {
	Label      string
	Shares     ShareholdingRawFormatted
	Percentage ShareholdingPercent
	Colors     ShareholdingColors
}

type ShareholdingRawFormatted struct {
	Raw       string
	Formatted string
}

type ShareholdingPercent struct {
	Raw       float64
	Formatted string
}

type ShareholdingColors struct {
	Light string
	Dark  string
}
