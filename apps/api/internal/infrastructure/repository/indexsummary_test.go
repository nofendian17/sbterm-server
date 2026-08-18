package repository

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nofendian17/sbterm/libs/stockbit"
)

const indexSummaryRepoBody = `{"data":{"cagr":"","change":6365.37,"drawdown":"","markingpoint":"{}","percentage":"0.00","xaxisopt":"intraday","previous":6409.654,"line_weight":1.6,"previous_timeframe_price":{"date":"","formatted_date":"2026-08-07","value":"6409.65","change":0},"chart_type":"PRICE_CHART_TYPE_LINE","interval_in_minutes":0,"allowed_chart_type":["PRICE_CHART_TYPE_LINE","PRICE_CHART_TYPE_CANDLE"],"max_candles":0,"prices":[{"date":"1786327200000","formatted_date":"2026-08-10 09:00:00","xlabel":"1","value":"6442.65","percentage":"0.03","change":2.048,"open":"","high":"","low":"","volume":""}]}}`

func TestIndexSummaryRepositoryGetIndexSummary(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr bool
	}{
		{
			name:   "returns mapped index summary",
			status: http.StatusOK,
			body:   indexSummaryRepoBody,
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
				assert.Equal(t, "/charts/IHSG/daily", r.URL.Path)
				assert.Equal(t, "2026-08-10", r.URL.Query().Get("from"))
				assert.Equal(t, "INTERVAL_CHART_MINUTELY", r.URL.Query().Get("interval"))
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			client := stockbit.New(stockbit.WithBaseURL(srv.URL))
			repo := NewIndexSummaryRepository(client)

			got, err := repo.GetIndexSummary(context.Background(), "IHSG", "2026-08-10", "2026-08-10", "INTERVAL_CHART_MINUTELY")
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "intraday", got.XAxisOpt)
			assert.Equal(t, float64(6409.654), got.Previous)
			assert.Equal(t, "2026-08-07", got.PreviousTimeframePrice.FormattedDate)
			require.Len(t, got.Prices, 1)
			p := got.Prices[0]
			assert.Equal(t, "2026-08-10 09:00:00", p.FormattedDate)
			assert.Equal(t, "6442.65", p.Value)
			assert.Equal(t, float64(2.048), p.Change)
		})
	}
}
