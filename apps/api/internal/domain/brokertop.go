package domain

// BrokerTopData is the top broker ranking over a session: the covered date and
// the per-broker list.
type BrokerTopData struct {
	Date BrokerTopDate
	List []BrokerTopItem
}

type BrokerTopDate struct {
	From string
	To   string
	Idx  string
}

// BrokerTopItem is one broker's totals. Monetary/volume fields are strings,
// matching the upstream's string serialization.
type BrokerTopItem struct {
	Code           string
	Name           string
	InvestorType   string
	TotalValue     string
	NetValue       string
	BuyValue       string
	SellValue      string
	TotalVolume    string
	TotalFrequency string
	Group          string
}
