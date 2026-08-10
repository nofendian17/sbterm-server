package stockbit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const chartDailyBody = `{"message":"Successfully Get Daily Price data","data":{"chartbit":[{"date":"2026-08-10","unixdate":1786294800,"open":990,"high":1075,"low":975,"close":985,"volume":1255876400,"foreignbuy":363995640500,"foreignsell":258659313500,"frequency":122452,"foreignflow":-1624835353268,"soxclose":189748508800000,"dividend":0,"value":1279403011000,"shareoutstanding":192638080000,"freq_analyzer":0.6839892103258433,"lot":0}]}}`

const chartIntradayBody = `{"message":"Successfully Get Intraday data","data":{"allow_decimal":0,"chartbit":[{"close":985,"datetime":"2026-08-10 16:14:00","foreign_buy":0,"foreign_sell":0,"frequency":"1","high":985,"low":985,"open":985,"symbol":"DSSA","unix_timestamp":"1786353240","value":62350500,"volume":"63300","lot":0}]}}`

func TestGetChartPrice(t *testing.T) {
	tests := []struct {
		name           string
		symbol, tframe string
		from, to       string
		limit          int
		body           string
		handler        func(t *testing.T, w http.ResponseWriter, r *http.Request)
		check          func(t *testing.T, resp *ChartPriceResponse)
	}{
		{
			name:   "returns daily OHLC bars",
			symbol: "DSSA", tframe: "daily",
			from: "2025-08-10", to: "2026-08-10", limit: 0,
			body: chartDailyBody,
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/chartbit/DSSA/price/daily", r.URL.Path)
				assert.Equal(t, "2025-08-10", r.URL.Query().Get("from"))
				assert.Equal(t, "2026-08-10", r.URL.Query().Get("to"))
				assert.Equal(t, "0", r.URL.Query().Get("limit"))
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(chartDailyBody))
			},
			check: func(t *testing.T, resp *ChartPriceResponse) {
				require.Len(t, resp.Data.Chartbit, 1)
				p := resp.Data.Chartbit[0]
				assert.Equal(t, "2026-08-10", p.Date)
				assert.Equal(t, int64(1786294800), p.Unixdate)
				assert.Equal(t, float64(985), p.Close)
				assert.Equal(t, float64(1075), p.High)
				assert.Equal(t, float64(258659313500), p.ForeignSell)
			},
		},
		{
			name:   "returns intraday bars",
			symbol: "DSSA", tframe: "intraday",
			from: "1786357735", to: "1785949199", limit: 0,
			body: chartIntradayBody,
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/chartbit/DSSA/price/intraday", r.URL.Path)
				assert.Equal(t, "1786357735", r.URL.Query().Get("from"))
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(chartIntradayBody))
			},
			check: func(t *testing.T, resp *ChartPriceResponse) {
				require.Len(t, resp.Data.Chartbit, 1)
				p := resp.Data.Chartbit[0]
				assert.Equal(t, "1786353240", p.UnixTimestamp)
				assert.Equal(t, "DSSA", p.Symbol)
				assert.Equal(t, float64(62350500), p.Value)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tt.handler(t, w, r)
			}))
			defer srv.Close()

			resp, err := New(WithBaseURL(srv.URL)).GetChartPrice(context.Background(), tt.symbol, tt.tframe, tt.from, tt.to, tt.limit)
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, resp)
			}
		})
	}
}
