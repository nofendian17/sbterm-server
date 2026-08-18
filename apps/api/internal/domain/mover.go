package domain

type MarketMover struct {
	Symbol         string
	Name           string
	Price          float64
	ChangeValue    float64
	ChangePercent  float64
	Value          float64
	Volume         float64
	Frequency      float64
	NetForeignBuy  float64
	NetForeignSell float64
	IEP            float64
	IEV            float64
	IEVAL          float64
	IEPChangePrev  float64
}
