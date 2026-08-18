package domain

type CompanyProfile struct {
	Background                      string
	History                         *ProfileHistory
	KeyExecutive                    *ProfileKeyExecutive
	Address                         []ProfileAddress
	Subsidiary                      []ProfileSubsidiary
	Beneficiary                     []ProfileBeneficiary
	Shareholder                     []ProfileShareholder
	ShareholderDirectorCommissioner []ProfileShareholder
	ShareholderNumbers              []ProfileShareholderNumber
	ShareholderOnePercent           []ProfileShareholder
}

type ProfileHistory struct {
	Amount       string
	Board        string
	Date         string
	Price        string
	Registrar    string
	Shares       string
	Underwriters []string
	FreeFloat    string
}

type ProfileKeyExecutive struct {
	Commissioner            []ProfileExecutive
	Director                []ProfileExecutive
	IndependentCommissioner []ProfileExecutive
}

type ProfileExecutive struct {
	ID    string
	Key   string
	Value string
}

type ProfileAddress struct {
	Office  string
	Phone   string
	Fax     string
	Email   []string
	Website string
	NPWP    string
}

type ProfileSubsidiary struct {
	Company    string
	Percentage string
	Types      string
	Value      string
}

type ProfileBeneficiary struct {
	Name string
}

type ProfileShareholder struct {
	ID             string
	Percentage     string
	Name           string
	Value          string
	Badges         []string
	Type           string
	Location       string
	Nationality    string
	Domicile       string
	Scripless      string
	Scrip          string
	ValueFormatted string
	Classification string
}

type ProfileShareholderNumber struct {
	ShareholderDate string
	TotalShare      string
	Change          int64
	ChangeFormatted string
}
