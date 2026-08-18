package stockbit

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

const findataFinancialV2Path = "/findata-view/v2/financials/%s"

// FindataFinancialResponse is the v2 financial report response. Data is fully
// structured JSON.
type FindataFinancialResponse struct {
	Message string               `json:"message"`
	Data    FindataFinancialData `json:"data"`
}

type FindataFinancialData struct {
	Currency        []string          `json:"currency"`
	DefaultCurrency string            `json:"default_currency"`
	RoundingValue   []int             `json:"rounding_value"`
	DataTables      FindataDataTables `json:"data_tables"`
}

type FindataDataTables struct {
	Periods      []string         `json:"periods"`
	Accounts     []FindataAccount `json:"accounts"`
	MaxShowLevel int              `json:"max_show_level"`
}

type FindataAccount struct {
	ID                int64            `json:"id"`
	Level             int              `json:"level"`
	Name              string           `json:"name"`
	Values            []string         `json:"values"`
	Accounts          []FindataAccount `json:"accounts"`
	IsTotalExist      bool             `json:"is_total_exist"`
	IsDefaultExpanded bool             `json:"is_default_expanded"`
	MaxShowLevel      int              `json:"max_show_level"`
}

// GetFindataFinancial returns the structured financial report for a symbol.
// The access token is attached automatically.
func (c *Client) GetFindataFinancial(ctx context.Context, symbol string, dataType, isPercentage, page, reportType, statementType int) (*FindataFinancialResponse, error) {
	q := url.Values{}
	q.Set("data_type", strconv.Itoa(dataType))
	q.Set("is_percentage", strconv.Itoa(isPercentage))
	q.Set("page", strconv.Itoa(page))
	q.Set("report_type", strconv.Itoa(reportType))
	q.Set("statement_type", strconv.Itoa(statementType))
	var out FindataFinancialResponse
	if err := c.Get(ctx, fmt.Sprintf(findataFinancialV2Path, symbol), q, &out); err != nil {
		return nil, fmt.Errorf("stockbit: findata financial %s: %w", symbol, err)
	}
	return &out, nil
}
