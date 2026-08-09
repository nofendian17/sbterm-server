package repository

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nofendian17/sbterm-server/internal/infrastructure/stockbit"
)

const keystatsRepoBody = `{"data":{"closure_fin_items_results":[{"keystats_name":"Current Valuation","fin_name_results":[{"fitem":{"id":"12148","name":"Current PE Ratio (Annualised)","value":"1,187.45"},"hidden_graph_ico":false,"is_new_update":false}]}],"financial_year_parent":{"financial_year_groups":[{"financial_year_values":[{"year":"2026","period_values":[{"period":"Q1","quarter_value":"(8 B)","year":"2026","is_new_update":false}],"annualised_value":"16 B","ttm_value":"26 B","is_new_update":false,"dividend":"-","payout_ratio":"-","dividend_yield":"-"}]}],"financial_year_groups_usd":[]},"stats":{"current_share_outstanding":"24.62 B","market_cap":"19,324 B","enterprise_value":"19,526 B","free_float":"33.17%"},"info":"","dividend_group":{"fitem_id":["21507"],"dividend_year_values":[{"period":2026,"dividend":"-","ex_date":"-","payment_date":"-"}]},"financial_report_currency":["IDR"]}}`

func TestKeystatsRepositoryGetKeystats(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr bool
	}{
		{
			name:   "returns mapped keystats",
			status: http.StatusOK,
			body:   keystatsRepoBody,
		},
		{
			name:    "propagates upstream error",
			status:  http.StatusInternalServerError,
			body:    `{"message":"boom"}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/keystats/ratio/v1/BUVA", r.URL.Path)
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			client := stockbit.New(stockbit.WithBaseURL(srv.URL))
			repo := NewKeystatsRepository(client)

			got, err := repo.GetKeystats(context.Background(), "BUVA", 10)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "24.62 B", got.Stats.CurrentShareOutstanding)
			assert.Equal(t, "Current Valuation", got.ClosureFinItemsResults[0].KeystatsName)
			assert.Equal(t, "2026", got.FinancialYearParent.FinancialYearGroups[0].FinancialYearValues[0].Year)
			assert.Equal(t, "IDR", got.FinancialReportCurrency[0])
		})
	}
}