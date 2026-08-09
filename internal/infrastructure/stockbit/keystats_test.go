package stockbit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const keystatsBody = `{"message":"Successfully retrieved keystats data","data":{"closure_fin_items_results":[{"keystats_name":"Current Valuation","fin_name_results":[{"fitem":{"id":"12148","name":"Current PE Ratio (Annualised)","value":"1,187.45"},"hidden_graph_ico":false,"is_new_update":false}]}],"financial_year_parent":{"financial_year_groups":[{"financial_year_values":[{"year":"2026","period_values":[{"period":"Q1","quarter_value":"(8 B)","year":"2026","is_new_update":false}],"annualised_value":"16 B","ttm_value":"26 B","is_new_update":false,"dividend":"-","payout_ratio":"-","dividend_yield":"-"}]}],"financial_year_groups_usd":[]},"stats":{"current_share_outstanding":"24.62 B","market_cap":"19,324 B","enterprise_value":"19,526 B","free_float":"33.17%"},"info":"","dividend_group":{"fitem_id":["21507"],"dividend_year_values":[{"period":2026,"dividend":"-","ex_date":"-","payment_date":"-"}]},"financial_report_currency":["IDR"]}}`

func TestGetKeystats(t *testing.T) {
	tests := []struct {
		name      string
		symbol    string
		yearLimit int
		handler   func(t *testing.T, w http.ResponseWriter, r *http.Request)
		check     func(t *testing.T, resp *KeystatsResponse)
	}{
		{
			name:      "returns keystats ratios",
			symbol:    "BUVA",
			yearLimit: 10,
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/keystats/ratio/v1/BUVA", r.URL.Path)
				assert.Equal(t, "10", r.URL.Query().Get("year_limit"))
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(keystatsBody))
			},
			check: func(t *testing.T, resp *KeystatsResponse) {
				require.Len(t, resp.Data.ClosureFinItemsResults, 1)
				g := resp.Data.ClosureFinItemsResults[0]
				assert.Equal(t, "Current Valuation", g.KeystatsName)
				assert.Equal(t, "1,187.45", g.FinNameResults[0].Fitem.Value)
				y := resp.Data.FinancialYearParent.FinancialYearGroups[0].FinancialYearValues[0]
				assert.Equal(t, "2026", y.Year)
				assert.Equal(t, "(8 B)", y.PeriodValues[0].QuarterValue)
				assert.Equal(t, "24.62 B", resp.Data.Stats.CurrentShareOutstanding)
				assert.Equal(t, int(2026), resp.Data.DividendGroup.DividendYearValues[0].Period)
				assert.Equal(t, "IDR", resp.Data.FinancialReportCurrency[0])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tt.handler(t, w, r)
			}))
			defer srv.Close()

			resp, err := New(WithBaseURL(srv.URL)).GetKeystats(context.Background(), tt.symbol, tt.yearLimit)
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, resp)
			}
		})
	}
}