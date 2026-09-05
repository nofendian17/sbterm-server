package domain

// CompanyProfile is the aggregate of one stock's profile: the 1:1 header
// row plus all normalized children. It is the read shape (GET) and also the
// write shape (admin save replaces the whole cluster in one transaction).
type CompanyProfile struct {
	Symbol             string
	Background         *string
	Board              *string
	ListingDate        *string
	ListingPrice       *string
	IpoAmount          *string
	ListedShares       *string
	FreeFloat          *string
	Registrar          *string
	Executives         []CompanyExecutive
	Holdings           []CompanyHolding
	ShareholderNumbers []CompanyShareholderNumber
	Subsidiaries       []CompanySubsidiary
	Addresses          []CompanyAddress
}

// CompanyExecutive mirrors company_executives. Kind is one of
// commissioner / director / independent_commissioner.
type CompanyExecutive struct {
	Kind       string
	Name       string
	Role       *string
	ExternalID *string
	Position   int
}

// CompanyHolding mirrors company_holdings. HolderGroup is one of
// shareholder / one_percent / director_commissioner / beneficiary.
type CompanyHolding struct {
	HolderGroup   string
	Name          string
	Percentage    *float64
	PercentageRaw *string
	AmountRaw     *string
	Badges        []string
	Position      int
}

// CompanyShareholderNumber mirrors company_shareholder_numbers (one row per
// reporting period).
type CompanyShareholderNumber struct {
	ShareholderDate string
	TotalShare      *string
	Change          *int64
	ChangeFormatted *string
}

// CompanySubsidiary mirrors company_subsidiaries.
type CompanySubsidiary struct {
	Name              string
	BusinessType      *string
	Location          *string
	CommercialYear    *string
	TotalAssets       *float64
	TotalAssetsRaw    *string
	Percentage        *float64
	PercentageRaw     *string
	OperationalStatus *string
	Period            *string
	Position          int
}

// CompanyAddress mirrors company_addresses.
type CompanyAddress struct {
	Office   *string
	Phone    *string
	Fax      *string
	Website  *string
	Npwp     *string
	Emails   []string
	Position int
}
