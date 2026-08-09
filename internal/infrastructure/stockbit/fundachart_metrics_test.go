package stockbit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const fundaChartMetricsBody = `{"message":"Successfully retrieved metrics","data":[{"fitem_id":18,"fitem_name":"Size","show_chart_icon":0,"child":[{"fitem_id":2892,"fitem_name":"Market Cap","show_chart_icon":0,"child":[{"fitem_id":12626,"fitem_name":"+2 PE Standard Deviation (1 Year)","show_chart_icon":0,"child":[]}]}]}]}`

func TestGetFundaChartMetrics(t *testing.T) {
	tests := []struct {
		name       string
		metricName string
		handler    func(t *testing.T, w http.ResponseWriter, r *http.Request)
		check      func(t *testing.T, resp *FundaChartMetricsResponse)
	}{
		{
			name:       "returns metric tree",
			metricName: "fundachart",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/fundachart/metrics", r.URL.Path)
				assert.Equal(t, "fundachart", r.URL.Query().Get("metric_name"))
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(fundaChartMetricsBody))
			},
			check: func(t *testing.T, resp *FundaChartMetricsResponse) {
				require.Len(t, resp.Data, 1)
				g := resp.Data[0]
				assert.Equal(t, int64(18), g.FitemID)
				assert.Equal(t, "Size", g.FitemName)
				require.Len(t, g.Child, 1)
				assert.Equal(t, "Market Cap", g.Child[0].FitemName)
				require.Len(t, g.Child[0].Child, 1)
				assert.Equal(t, "+2 PE Standard Deviation (1 Year)", g.Child[0].Child[0].FitemName)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tt.handler(t, w, r)
			}))
			defer srv.Close()

			resp, err := New(WithBaseURL(srv.URL)).GetFundaChartMetrics(context.Background(), tt.metricName)
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, resp)
			}
		})
	}
}