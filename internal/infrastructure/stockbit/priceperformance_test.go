package stockbit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const pricePerformanceBody = `{"message":"Successfully retrieved price performance","data":{"prices":[{"close":{"raw":785,"formatted":"785"},"high":{"raw":805,"formatted":"805"},"low":{"raw":780,"formatted":"780"},"percentage":{"raw":-1.2578616,"formatted":"(-1.26%)"},"timeframe":"1D"},{"close":{"raw":785,"formatted":"785"},"high":{"raw":2320,"formatted":"2,320"},"low":{"raw":224,"formatted":"224"},"percentage":{"raw":250.44643,"formatted":"(+250.45%)"},"timeframe":"1Y"}]}}`

func TestGetPricePerformance(t *testing.T) {
	tests := []struct {
		name    string
		symbol  string
		handler func(t *testing.T, w http.ResponseWriter, r *http.Request)
		check   func(t *testing.T, resp *PricePerformanceResponse)
	}{
		{
			name:   "returns per-timeframe performance",
			symbol: "BUVA",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/company-price-feed/price-performance/BUVA", r.URL.Path)
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(pricePerformanceBody))
			},
			check: func(t *testing.T, resp *PricePerformanceResponse) {
				require.Len(t, resp.Data.Prices, 2)
				p := resp.Data.Prices[0]
				assert.Equal(t, "1D", p.Timeframe)
				assert.Equal(t, float64(785), p.Close.Raw)
				assert.Equal(t, float64(-1.2578616), p.Percentage.Raw)
				last := resp.Data.Prices[1]
				assert.Equal(t, "1Y", last.Timeframe)
				assert.Equal(t, float64(250.44643), last.Percentage.Raw)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tt.handler(t, w, r)
			}))
			defer srv.Close()

			resp, err := New(WithBaseURL(srv.URL)).GetPricePerformance(context.Background(), tt.symbol)
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, resp)
			}
		})
	}
}