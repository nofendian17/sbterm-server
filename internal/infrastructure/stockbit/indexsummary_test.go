package stockbit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// indexSummaryBody mirrors the real upstream response captured from
// /charts/IHSG/daily?interval=INTERVAL_CHART_MINUTELY (see /tmp/ihsg_chart_cap.json).
const indexSummaryBody = `{"message":"Successfully retrieved company daily chart","data":{"cagr":"","change":6365.37,"drawdown":"","markingpoint":"{}","percentage":"0.00","timeframe":"","xaxisopt":"intraday","previous":6409.654,"line_weight":1.6,"previous_timeframe_price":{"date":"","formatted_date":"2026-08-07","xlabel":"","value":"6409.65","percentage":"","change":0,"open":"","high":"","low":"","volume":""},"chart_type":"PRICE_CHART_TYPE_LINE","interval_in_minutes":0,"allowed_chart_type":["PRICE_CHART_TYPE_LINE","PRICE_CHART_TYPE_CANDLE"],"max_candles":0,"prices":[{"date":"1786327200000","formatted_date":"2026-08-10 09:00:00","xlabel":"1","value":"6442.65","percentage":"0.03","change":2.048,"open":"","high":"","low":"","volume":""},{"date":"0","formatted_date":"2026-08-10 16:14:00","xlabel":"334","value":"6365.37","percentage":"-1.17","change":-75.223,"open":"","high":"","low":"","volume":""}]}}`

func TestGetIndexSummary(t *testing.T) {
	tests := []struct {
		name             string
		symbol, from, to string
		interval         string
		body             string
		handler          func(t *testing.T, w http.ResponseWriter, r *http.Request)
		check            func(t *testing.T, resp *IndexSummaryResponse)
	}{
		{
			name:     "returns minutely index summary",
			symbol:   "IHSG",
			from:     "2026-08-10",
			to:       "2026-08-10",
			interval: "INTERVAL_CHART_MINUTELY",
			body:     indexSummaryBody,
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/charts/IHSG/daily", r.URL.Path)
				assert.Equal(t, "2026-08-10", r.URL.Query().Get("from"))
				assert.Equal(t, "2026-08-10", r.URL.Query().Get("to"))
				assert.Equal(t, "INTERVAL_CHART_MINUTELY", r.URL.Query().Get("interval"))
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(indexSummaryBody))
			},
			check: func(t *testing.T, resp *IndexSummaryResponse) {
				require.Len(t, resp.Data.Prices, 2)
				assert.Equal(t, "intraday", resp.Data.XAxisOpt)
				assert.Equal(t, float64(6409.654), float64(resp.Data.Previous))
				assert.Equal(t, "2026-08-07", resp.Data.PreviousTimeframePrice.FormattedDate)
				p := resp.Data.Prices[0]
				assert.Equal(t, "2026-08-10 09:00:00", p.FormattedDate)
				assert.Equal(t, "6442.65", p.Value)
				assert.Equal(t, "0.03", p.Percentage)
				assert.Equal(t, float64(2.048), float64(p.Change))
			},
		},
		{
			name:   "omits interval when empty",
			symbol: "IHSG",
			from:   "2026-08-10",
			to:     "2026-08-10",
			body:   indexSummaryBody,
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/charts/IHSG/daily", r.URL.Path)
				assert.Equal(t, "", r.URL.Query().Get("interval"))
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(indexSummaryBody))
			},
			check: func(t *testing.T, resp *IndexSummaryResponse) {
				require.Len(t, resp.Data.Prices, 2)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tt.handler(t, w, r)
			}))
			defer srv.Close()

			resp, err := New(WithBaseURL(srv.URL)).GetIndexSummary(context.Background(), tt.symbol, tt.from, tt.to, tt.interval)
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, resp)
			}
		})
	}
}
