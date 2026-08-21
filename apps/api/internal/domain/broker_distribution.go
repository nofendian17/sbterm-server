package domain

// BrokerDistributionData is the per-broker distribution of one symbol.
type BrokerDistributionData struct {
	DateInfo  string
	ByValue   BrokerDistributionSides
	ByVolume  BrokerDistributionSides
	StartDate string
	EndDate   string
}

// BrokerDistributionSides holds the top buy/sell brokers. The side selected by
// the request's data_type is populated; the other one stays empty.
type BrokerDistributionSides struct {
	TopBrokerBuy  []BrokerDistributionEntry
	TopBrokerSell []BrokerDistributionEntry
}

type BrokerDistributionEntry struct {
	Detail       BrokerDistributionCounterparty
	DistributeTo []BrokerDistributionCounterparty
}

type BrokerDistributionCounterparty struct {
	Code   string
	Type   string // Lokal / Asing / Pemerintah
	Amount int64
}
