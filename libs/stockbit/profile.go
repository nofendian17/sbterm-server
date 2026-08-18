package stockbit

import (
	"context"
	"encoding/json"
	"fmt"
)

const companyProfilePath = "/emitten/%s/profile"

// CompanyProfileResponse is the company profile response: data is the profile
// itself.
type CompanyProfileResponse struct {
	Data CompanyProfile `json:"data"`
}

// CompanyProfile captures the full upstream profile payload. Selected sections
// are typed; the remaining fund-only sections are preserved verbatim as raw
// JSON so nothing upstream adds is dropped at the infrastructure layer.
type CompanyProfile struct {
	Background                      string                        `json:"background"`
	History                         *ProfileHistory               `json:"history"`
	KeyExecutive                    *ProfileKeyExecutive          `json:"key_executive"`
	Address                         []ProfileAddress              `json:"address"`
	Subsidiary                      []ProfileSubsidiary           `json:"subsidiary"`
	Beneficiary                     []ProfileBeneficiary          `json:"beneficiary"`
	Shareholder                     []ProfileShareholder          `json:"shareholder"`
	ShareholderDirectorCommissioner []ProfileShareholder          `json:"shareholder_director_commissioner"`
	ShareholderNumbers              []ProfileShareholderNumber    `json:"shareholder_numbers"`
	ShareholderOnePercent           *ProfileShareholderOnePercent `json:"shareholder_one_percent"`
	Badges                          []string                      `json:"badges"`
	ListingInformation              *ProfileListingInformation    `json:"listing_information"`
	Profile                         *ProfileFundInfo              `json:"profile"`
	Secretary                       []ProfileSecretary            `json:"secretary"`
	AssetAllocation                 []json.RawMessage             `json:"asset_allocation"`
	Classification                  json.RawMessage               `json:"classification"`
	Fee                             []json.RawMessage             `json:"fee"`
	PDF                             []json.RawMessage             `json:"pdf"`
	ShareholderReksa                []json.RawMessage             `json:"shareholder_reksa"`
	TopHoldings                     []json.RawMessage             `json:"top_holdings"`
}

type ProfileHistory struct {
	Amount       string   `json:"amount"`
	Board        string   `json:"board"`
	Date         string   `json:"date"`
	Price        string   `json:"price"`
	Registrar    string   `json:"registrar"`
	Shares       string   `json:"shares"`
	Underwriters []string `json:"underwriters"`
	FreeFloat    string   `json:"free_float"`
}

type ProfileKeyExecutive struct {
	Commissioner            []ProfileExecutive `json:"commissioner"`
	Director                []ProfileExecutive `json:"director"`
	IndependentCommissioner []ProfileExecutive `json:"independent_commissioner"`
}

type ProfileExecutive struct {
	ID    string `json:"id"`
	Key   string `json:"key"`
	Value string `json:"value"`
}

type ProfileAddress struct {
	Office  string   `json:"office"`
	Phone   string   `json:"phone"`
	Fax     string   `json:"fax"`
	Email   []string `json:"email"`
	Website string   `json:"website"`
	NPWP    string   `json:"npwp"`
}

type ProfileSubsidiary struct {
	Company    string `json:"company"`
	Percentage string `json:"percentage"`
	Types      string `json:"types"`
	Value      string `json:"value"`
}

type ProfileBeneficiary struct {
	Name string `json:"name"`
}

type ProfileShareholder struct {
	ID             string   `json:"id"`
	Percentage     string   `json:"percentage"`
	Name           string   `json:"name"`
	Value          string   `json:"value"`
	Badges         []string `json:"badges"`
	Type           string   `json:"type"`
	Location       string   `json:"location"`
	Nationality    string   `json:"nationality"`
	Domicile       string   `json:"domicile"`
	Scripless      string   `json:"scripless"`
	Scrip          string   `json:"scrip"`
	ValueFormatted string   `json:"value_formatted"`
	Classification string   `json:"classification"`
}

type ProfileShareholderOnePercent struct {
	Shareholder []ProfileShareholder `json:"shareholder"`
}

type ProfileShareholderNumber struct {
	ShareholderDate string `json:"shareholder_date"`
	TotalShare      string `json:"total_share"`
	Change          int64  `json:"change"`
	ChangeFormatted string `json:"change_formatted"`
	ChangeValue     string `json:"change_value"`
}

type ProfileListingInformation struct {
	ExerciseStartDate  string              `json:"exercise_start_date"`
	ExerciseEndDate    string              `json:"exercise_end_date"`
	ExercisePrice      int64               `json:"exercise_price"`
	ExpireDate         string              `json:"expire_date"`
	ListingDate        string              `json:"listing_date"`
	ForeignPercentage  ProfileRawFormatted `json:"foreign_percentage"`
	LocalPercentage    ProfileRawFormatted `json:"local_percentage"`
	NumberOfSecurities int64               `json:"number_of_securities"`
	TotalShares        int64               `json:"total_shares"`
}

type ProfileRawFormatted struct {
	Raw       int64  `json:"raw"`
	Formatted string `json:"formatted"`
}

type ProfileValueInfo struct {
	Value string `json:"value"`
	Info  string `json:"info"`
}

// ProfileFundInfo carries the fund-only profile block (empty for equities).
// It is captured as-is so the structure survives at the infrastructure layer.
type ProfileFundInfo struct {
	FundType           ProfileValueInfo  `json:"fund_type"`
	InceptionDate      string            `json:"inception_date"`
	FundManager        string            `json:"fund_manager"`
	FundManagerIco     string            `json:"fund_manager_ico"`
	CustodianBank      string            `json:"custodian_bank"`
	CustodianIco       string            `json:"custodian_ico"`
	RiskLevel          ProfileValueInfo  `json:"risk_level"`
	AUM                ProfileValueInfo  `json:"aum"`
	MaxDrawdown        ProfileValueInfo  `json:"maxdrawdown"`
	CAGR5Year          ProfileValueInfo  `json:"cagr5year"`
	ExpenseRatio       ProfileValueInfo  `json:"expense_ratio"`
	AverageYield       ProfileValueInfo  `json:"average_yield"`
	Prospectus         ProfileProspectus `json:"prospectus"`
	FundFactSheet      []json.RawMessage `json:"fund_fact_sheet"`
	RedemptionBankName string            `json:"redemption_bank_name"`
	MinBuy             string            `json:"min_buy"`
	BuyFee             string            `json:"buy_fee"`
	SellFee            string            `json:"sell_fee"`
}

type ProfileProspectus struct {
	Name string `json:"name"`
	File string `json:"file"`
	Dir  string `json:"dir"`
	URL  string `json:"url"`
}

type ProfileSecretary struct {
	ID         string `json:"id"`
	Key        string `json:"key"`
	LastUpdate string `json:"lastupdate"`
	Value      string `json:"value"`
}

// GetProfile returns the curated sections of a company profile. The access
// token is attached automatically.
func (c *Client) GetProfile(ctx context.Context, symbol string) (*CompanyProfileResponse, error) {
	var out CompanyProfileResponse
	if err := c.Get(ctx, fmt.Sprintf(companyProfilePath, symbol), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
