package stockbit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const historicalSummaryBody = `{"message":"Successfully loaded historical summary","data":{"result":[{"date":"2026-08-10","close":945,"change":-30,"value":1928786961000,"volume":19414009,"frequency":184265,"foreign_buy":421517824000,"foreign_sell":453168550000,"net_foreign":-31650726000,"open":990,"high":1075,"low":920,"average":994,"change_percentage":-3.08},{"date":"2026-08-03","close":975,"change":125,"value":3018692992500,"volume":32167874,"frequency":278462,"foreign_buy":825006183000,"foreign_sell":522977004500,"net_foreign":302029178500,"open":850,"high":1060,"low":830,"average":938,"change_percentage":14.71}],"paginate":{"next_page":"2"}}}`

func TestGetHistoricalSummary(t *testing.T) {
	tests := []struct {
		name      string
		symbol    string
		period    string
		startDate string
		endDate   string
		limit     int
		page      int
		handler   func(t *testing.T, w http.ResponseWriter, r *http.Request)
		check     func(t *testing.T, resp *HistoricalSummaryResponse)
	}{
		{
			name:      "returns historical summary with all params",
			symbol:    "DSSA",
			period:    "HS_PERIOD_WEEKLY",
			startDate: "2025-08-11",
			endDate:   "2026-08-11",
			limit:     12,
			page:      1,
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/company-price-feed/historical/summary/DSSA", r.URL.Path)
				q := r.URL.Query()
				assert.Equal(t, "HS_PERIOD_WEEKLY", q.Get("period"))
				assert.Equal(t, "2025-08-11", q.Get("start_date"))
				assert.Equal(t, "2026-08-11", q.Get("end_date"))
				assert.Equal(t, "12", q.Get("limit"))
				assert.Equal(t, "1", q.Get("page"))
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(historicalSummaryBody))
			},
			check: func(t *testing.T, resp *HistoricalSummaryResponse) {
				d := resp.Data
				require.Len(t, d.Result, 2)
				assert.Equal(t, "2", d.Paginate.NextPage)
				first := d.Result[0]
				assert.Equal(t, "2026-08-10", first.Date)
				assert.Equal(t, 945.0, first.Close)
				assert.Equal(t, -30.0, first.Change)
				assert.Equal(t, int64(1928786961000), first.Value)
				assert.Equal(t, int64(19414009), first.Volume)
				assert.Equal(t, int64(184265), first.Frequency)
				assert.Equal(t, int64(421517824000), first.ForeignBuy)
				assert.Equal(t, int64(453168550000), first.ForeignSell)
				assert.Equal(t, int64(-31650726000), first.NetForeign)
				assert.Equal(t, 990.0, first.Open)
				assert.Equal(t, 1075.0, first.High)
				assert.Equal(t, 920.0, first.Low)
				assert.Equal(t, 994.0, first.Average)
				assert.Equal(t, -3.08, first.ChangePercentage)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tt.handler(t, w, r)
			}))
			defer srv.Close()

			resp, err := New(WithBaseURL(srv.URL)).GetHistoricalSummary(
				context.Background(), tt.symbol, tt.period, tt.startDate, tt.endDate, tt.limit, tt.page)
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, resp)
			}
		})
	}
}
