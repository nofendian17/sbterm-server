package stockbit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const findataFinancialV2Body = `{"message":"Successfully retrieved company financial","data":{"currency":["IDR","USD"],"default_currency":"IDR","html_report":"","rounding_value":[1000000000,1000000],"data_tables":{"periods":["12M 2025","12M 2024"],"accounts":[{"id":190,"level":1,"name":"<b>Arus Kas Dari Aktivitas Operasi</b>","values":[],"accounts":[{"id":191,"level":2,"name":"Penerimaan Kas Dari Pelanggan","values":["132,751 B","37,651 B"],"accounts":[],"is_total_exist":true,"is_default_expanded":false,"max_show_level":1}],"is_total_exist":true,"is_default_expanded":false,"max_show_level":1}],"max_show_level":1}}}`

func TestGetFindataFinancialV2(t *testing.T) {
	tests := []struct {
		name           string
		symbol         string
		dataType       int
		isPercentage   int
		page           int
		reportType     int
		statementType  int
		handler        func(t *testing.T, w http.ResponseWriter, r *http.Request)
		check          func(t *testing.T, resp *FindataFinancialResponse)
	}{
		{
			name:          "returns structured financial report",
			symbol:        "BRPT",
			dataType:      1,
			isPercentage:  0,
			page:          1,
			reportType:    3,
			statementType: 2,
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/findata-view/v2/financials/BRPT", r.URL.Path)
				assert.Equal(t, "1", r.URL.Query().Get("data_type"))
				assert.Equal(t, "0", r.URL.Query().Get("is_percentage"))
				assert.Equal(t, "1", r.URL.Query().Get("page"))
				assert.Equal(t, "3", r.URL.Query().Get("report_type"))
				assert.Equal(t, "2", r.URL.Query().Get("statement_type"))
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(findataFinancialV2Body))
			},
			check: func(t *testing.T, resp *FindataFinancialResponse) {
				assert.Equal(t, "IDR", resp.Data.DefaultCurrency)
				require.Len(t, resp.Data.DataTables.Periods, 2)
				assert.Equal(t, "12M 2025", resp.Data.DataTables.Periods[0])
				require.Len(t, resp.Data.DataTables.Accounts, 1)
				a := resp.Data.DataTables.Accounts[0]
				assert.Equal(t, int64(190), a.ID)
				assert.Equal(t, 1, a.Level)
				require.Len(t, a.Accounts, 1)
				child := a.Accounts[0]
				assert.Equal(t, "Penerimaan Kas Dari Pelanggan", child.Name)
				assert.Equal(t, "132,751 B", child.Values[0])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tt.handler(t, w, r)
			}))
			defer srv.Close()

			resp, err := New(WithBaseURL(srv.URL)).GetFindataFinancial(context.Background(), tt.symbol, tt.dataType, tt.isPercentage, tt.page, tt.reportType, tt.statementType)
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, resp)
			}
		})
	}
}