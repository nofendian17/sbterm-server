package domain

type SubsidiaryData struct {
	Currency          string
	LastUpdatedPeriod string
	Unit              string
	Subsidiaries      []Subsidiary
}

type Subsidiary struct {
	CompanyName       string
	BusinessType      string
	Location          string
	CommercialYear    string
	TotalAssets       string
	Percentage        string
	ID                int64
	OperationalStatus string
	Period            string
	Raw               *string
}
