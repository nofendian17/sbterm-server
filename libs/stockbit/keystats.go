package stockbit

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

const keystatsPath = "/keystats/ratio/v1/%s"

// KeystatsResponse is the key-stats ratio response: data holds the full
// financial summary.
type KeystatsResponse struct {
	Message string       `json:"message"`
	Data    KeystatsData `json:"data"`
}

type KeystatsData struct {
	ClosureFinItemsResults  []KeystatsFinGroup    `json:"closure_fin_items_results"`
	FinancialYearParent     KeystatsYearParent    `json:"financial_year_parent"`
	Stats                   KeystatsStats         `json:"stats"`
	Info                    string                `json:"info"`
	DividendGroup           KeystatsDividendGroup `json:"dividend_group"`
	FinancialReportCurrency []string              `json:"financial_report_currency"`
}

type KeystatsFinGroup struct {
	KeystatsName   string         `json:"keystats_name"`
	FinNameResults []KeystatsItem `json:"fin_name_results"`
}

type KeystatsItem struct {
	Fitem          KeystatsFitem `json:"fitem"`
	IsNewUpdate    bool          `json:"is_new_update"`
	HiddenGraphIco bool          `json:"hidden_graph_ico"`
}

type KeystatsFitem struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

type KeystatsYearParent struct {
	FinancialYearGroups    []KeystatsYearGroup `json:"financial_year_groups"`
	FinancialYearGroupsUSD []KeystatsYearGroup `json:"financial_year_groups_usd"`
}

type KeystatsYearGroup struct {
	FinancialYearValues []KeystatsYear `json:"financial_year_values"`
}

type KeystatsYear struct {
	Year            string           `json:"year"`
	PeriodValues    []KeystatsPeriod `json:"period_values"`
	AnnualisedValue string           `json:"annualised_value"`
	TTMValue        string           `json:"ttm_value"`
	IsNewUpdate     bool             `json:"is_new_update"`
	Dividend        string           `json:"dividend"`
	PayoutRatio     string           `json:"payout_ratio"`
	DividendYield   string           `json:"dividend_yield"`
}

type KeystatsPeriod struct {
	Period       string `json:"period"`
	Year         string `json:"year"`
	QuarterValue string `json:"quarter_value"`
	IsNewUpdate  bool   `json:"is_new_update"`
}

type KeystatsStats struct {
	CurrentShareOutstanding string `json:"current_share_outstanding"`
	MarketCap               string `json:"market_cap"`
	EnterpriseValue         string `json:"enterprise_value"`
	FreeFloat               string `json:"free_float"`
}

type KeystatsDividendGroup struct {
	FitemID            []string               `json:"fitem_id"`
	DividendYearValues []KeystatsDividendYear `json:"dividend_year_values"`
}

type KeystatsDividendYear struct {
	Period      int    `json:"period"`
	Dividend    string `json:"dividend"`
	ExDate      string `json:"ex_date"`
	PaymentDate string `json:"payment_date"`
}

// GetKeystats returns the key-stats ratio for a symbol. The access token is
// attached automatically.
func (c *Client) GetKeystats(ctx context.Context, symbol string, yearLimit int) (*KeystatsResponse, error) {
	q := url.Values{}
	if yearLimit > 0 {
		q.Set("year_limit", strconv.Itoa(yearLimit))
	}
	var out KeystatsResponse
	if err := c.Get(ctx, fmt.Sprintf(keystatsPath, symbol), q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
