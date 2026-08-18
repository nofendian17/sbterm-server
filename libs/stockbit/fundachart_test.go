package stockbit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const fundaChartBody = `{"message":"Successfully retrieved chart data","data":[{"company_id":105,"company_name":"BUVA","ratios":[{"decimal_point":2,"group_data":false,"item_id":12148,"item_name":"Current PE Ratio (Annualised)","item_type":6,"suffix":"","xaxis_id":7,"yaxis_id":3,"chart_data":[{"date":1470762000,"formated_date":"2016-08-10","value":-31.62,"ratio_value":-31.62}]}]}]}`

func TestGetFundaChart(t *testing.T) {
	tests := []struct {
		name      string
		symbol    string
		item      string
		timeframe string
		handler   func(t *testing.T, w http.ResponseWriter, r *http.Request)
		check     func(t *testing.T, resp *FundaChartResponse)
	}{
		{
			name:      "returns raw ratio series",
			symbol:    "BUVA",
			item:      "12148",
			timeframe: "10y",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/fundachart", r.URL.Path)
				assert.Equal(t, "BUVA", r.URL.Query().Get("companies"))
				assert.Equal(t, "12148", r.URL.Query().Get("item"))
				assert.Equal(t, "10y", r.URL.Query().Get("timeframe"))
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(fundaChartBody))
			},
			check: func(t *testing.T, resp *FundaChartResponse) {
				require.Len(t, resp.Data, 1)
				c := resp.Data[0]
				assert.Equal(t, int64(105), c.CompanyID)
				assert.Equal(t, "BUVA", c.CompanyName)
				require.Len(t, c.Ratios, 1)
				rt := c.Ratios[0]
				assert.Equal(t, int64(12148), rt.ItemID)
				require.Len(t, rt.ChartData, 1)
				p := rt.ChartData[0]
				assert.Equal(t, int64(1470762000), p.Date)
				assert.Equal(t, "2016-08-10", p.FormatedDate)
				assert.Equal(t, float64(-31.62), p.Value)
			},
		},
		{
			name:      "forwards comma-separated items",
			symbol:    "BUVA",
			item:      "2661,2525,1562",
			timeframe: "3y",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "2661,2525,1562", r.URL.Query().Get("item"))
				assert.Equal(t, "3y", r.URL.Query().Get("timeframe"))
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(fundaChartBody))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tt.handler(t, w, r)
			}))
			defer srv.Close()

			resp, err := New(WithBaseURL(srv.URL)).GetFundaChart(context.Background(), tt.symbol, tt.item, tt.timeframe)
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, resp)
			}
		})
	}
}
