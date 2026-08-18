package domain

// BrokerTopData is the top broker ranking over a session: the covered date and
// the per-broker list.
type BrokerTopData struct {
	Date BrokerTopDate   `json:"date"`
	List []BrokerTopItem `json:"list"`
}

type BrokerTopDate struct {
	From string `json:"from"`
	To   string `json:"to"`
	Idx  string `json:"idx"`
}

// BrokerTopItem is one broker's totals. Monetary/volume fields are strings,
// matching the upstream's string serialization.
type BrokerTopItem struct {
	Code           string `json:"code"`
	Name           string `json:"name"`
	InvestorType   string `json:"investor_type"`
	TotalValue     string `json:"total_value"`
	NetValue       string `json:"net_value"`
	BuyValue       string `json:"buy_value"`
	SellValue      string `json:"sell_value"`
	TotalVolume    string `json:"total_volume"`
	TotalFrequency string `json:"total_frequency"`
	Group          string `json:"group"`
}
