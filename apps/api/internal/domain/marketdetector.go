package domain

type MarketDetectorData struct {
	BandarDetector BandarDetector
	BrokerSummary  BrokerSummary
	From           string
	To             string
}

type BandarDetector struct {
	Average             float64
	Avg                 BandarAccdist
	Avg5                BandarAccdist
	BrokerAccdist       string
	NumberBrokerBuysell int
	Top1                BandarAccdist
	Top3                BandarAccdist
	Top5                BandarAccdist
	Top10               BandarAccdist
	TotalBuyer          int
	TotalSeller         int
	Value               int64
	Volume              int64
}

type BandarAccdist struct {
	Accdist string
	Amount  int64
	Percent float64
	Vol     float64
}

type BrokerSummary struct {
	BrokersBuy  []BrokerBuy
	BrokersSell []BrokerSell
	Symbol      string
}

type BrokerBuy struct {
	Blot             string
	Blotv            string
	Bval             string
	Bvalv            string
	NetbsBrokerCode  string
	NetbsBuyAvgPrice string
	NetbsDate        string
	NetbsStockCode   string
	Type             string
	Freq             string
}

type BrokerSell struct {
	NetbsBrokerCode   string
	NetbsDate         string
	NetbsSellAvgPrice string
	NetbsStockCode    string
	Slot              string
	Slotv             string
	Sval              string
	Svalv             string
	Type              string
	Freq              string
}
