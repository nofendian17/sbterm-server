package domain

type MarketSession struct {
	Datetime string
	FCA      MarketSessionSegment
	Regular  MarketSessionSegment
}

type MarketSessionSegment struct {
	StateName      string
	IsLastSession  bool
	IsEndOfDay     bool
	StateStartTime string
	StateEndTime   string
	TimeLeft       string
}
