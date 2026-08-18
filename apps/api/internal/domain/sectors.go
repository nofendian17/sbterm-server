package domain

type Sector struct {
	Symbol    string
	ID        string
	Icon      string
	Type      string
	Last      float64
	Change    string
	Percent   float64
	Companies []SubsectorCompany
}
