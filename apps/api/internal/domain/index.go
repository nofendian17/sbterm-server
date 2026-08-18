package domain

type Indexes struct {
	Main []Index
	All  []Index
}

type Index struct {
	Symbol    string
	Name      string
	Last      string
	Change    string
	Percent   string
	MarketCap string
}
