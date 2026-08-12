package stockbit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// activityHistoricalBody mirrors the real upstream response captured from
// /order-trade/broker/activity/historical (see /tmp/stockbit_activity_historical.json),
// trimmed to two daily records and the monthly summary.
const activityHistoricalBody = `{"message":"Successfully loaded broker activity historical data","data":{"date_from":"2026-07-01","date_to":"2026-08-12","symbols":["CUAN"],"broker_codes":["ZP","BK"],"broker_name":"","records":[{"date":"2026-08-12","broker_code":"","trade_activity":{"net_summary":{"avg_price":796.7837407992519,"freq":4947,"lot":-141664,"value":-11740235500},"buy_summary":{"avg_price":786.0812149497357,"freq":1937,"lot":422964,"value":33248405500},"sell_summary":{"avg_price":796.7837407992519,"freq":4947,"lot":564628,"value":44988641000},"foreign_summary":{"foreign_buy":0,"foreign_sell":0,"net_foreign":0},"total_buy_lot":{"amount":422964,"pct":42.82780743464913},"total_sell_lot":{"amount":564628,"pct":57.172192565350876}},"price_activity":{"close_price":"870","return_summary":{"amount":73.21625920074814,"pct":8.415661977097487}}},{"date":"2026-08-11","broker_code":"","trade_activity":{"net_summary":{"avg_price":687.0822704832468,"freq":531,"lot":51824,"value":3522402000},"buy_summary":{"avg_price":687.0822704832468,"freq":531,"lot":127409,"value":8754046500},"sell_summary":{"avg_price":692.1538003572138,"freq":722,"lot":75585,"value":5231644500},"foreign_summary":{"foreign_buy":0,"foreign_sell":0,"net_foreign":0},"total_buy_lot":{"amount":127409,"pct":62.76490930766426},"total_sell_lot":{"amount":75585,"pct":37.23509069233573}},"price_activity":{"close_price":"720","return_summary":{"amount":32.917729516753184,"pct":4.571906877326831}}}],"pagination":{"page":1,"limit":100,"has_next":false,"has_prev":false},"summary":{"group_type":"INTERVAL_TYPE_MONTHLY","data":[{"date_from":"2026-08-01","date_to":"2026-08-12","net_summary":{"avg_price":731.6727956504039,"freq":13735,"lot":-514137,"value":-36687700500}},{"date_from":"2026-07-01","date_to":"2026-07-31","net_summary":{"avg_price":676.1897003304894,"freq":24248,"lot":7726,"value":-1823597000}}]}}}`

func TestGetActivityHistorical(t *testing.T) {
	tests := []struct {
		name         string
		interval     string
		dateFrom     string
		dateTo       string
		brokerCodes  []string
		symbols      []string
		marketBoard  string
		investorType string
		netInterval  string
		handler      func(t *testing.T, w http.ResponseWriter, r *http.Request)
		check        func(t *testing.T, resp *ActivityHistoricalResponse)
	}{
		{
			name:         "returns historical activity with all params",
			interval:     "INTERVAL_DAILY",
			dateFrom:     "2026-07-01",
			dateTo:       "2026-08-31",
			brokerCodes:  []string{"ZP", "BK"},
			symbols:      []string{"CUAN"},
			marketBoard:  "BOARD_TYPE_REGULAR",
			investorType: "INVESTOR_TYPE_ALL",
			netInterval:  "INTERVAL_MONTHLY",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/order-trade/broker/activity/historical", r.URL.Path)
				q := r.URL.Query()
				assert.Equal(t, "INTERVAL_DAILY", q.Get("interval"))
				assert.Equal(t, "2026-07-01", q.Get("date_from"))
				assert.Equal(t, "2026-08-31", q.Get("date_to"))
				assert.Equal(t, []string{"ZP", "BK"}, q["broker_codes"])
				assert.Equal(t, []string{"CUAN"}, q["symbols"])
				assert.Equal(t, "BOARD_TYPE_REGULAR", q.Get("market_board"))
				assert.Equal(t, "INVESTOR_TYPE_ALL", q.Get("investor_type"))
				assert.Equal(t, "INTERVAL_MONTHLY", q.Get("net_interval"))
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(activityHistoricalBody))
			},
			check: func(t *testing.T, resp *ActivityHistoricalResponse) {
				d := resp.Data
				assert.Equal(t, "2026-07-01", d.DateFrom)
				assert.Equal(t, "2026-08-12", d.DateTo)
				assert.Equal(t, []string{"CUAN"}, d.Symbols)
				assert.Equal(t, []string{"ZP", "BK"}, d.BrokerCodes)
				require.Len(t, d.Records, 2)
				rec := d.Records[0]
				assert.Equal(t, "2026-08-12", rec.Date)
				assert.Equal(t, 796.7837407992519, rec.TradeActivity.NetSummary.AveragePrice)
				assert.Equal(t, -141664.0, rec.TradeActivity.NetSummary.Lot)
				assert.Equal(t, 0.0, rec.TradeActivity.ForeignSummary.NetForeign)
				assert.Equal(t, 42.82780743464913, rec.TradeActivity.TotalBuyLot.Pct)
				assert.Equal(t, "870", rec.PriceActivity.ClosePrice)
				assert.Equal(t, 8.415661977097487, rec.PriceActivity.ReturnSummary.Pct)
				assert.False(t, d.Pagination.HasNext)
				assert.Equal(t, 1, d.Pagination.Page)
				assert.Equal(t, "INTERVAL_TYPE_MONTHLY", d.Summary.GroupType)
				require.Len(t, d.Summary.Data, 2)
				assert.Equal(t, "2026-07-01", d.Summary.Data[1].DateFrom)
				assert.Equal(t, -1823597000.0, d.Summary.Data[1].NetSummary.Value)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tt.handler(t, w, r)
			}))
			defer srv.Close()

			resp, err := New(WithBaseURL(srv.URL)).GetActivityHistorical(context.Background(), tt.interval, tt.dateFrom, tt.dateTo, tt.brokerCodes, tt.symbols, tt.marketBoard, tt.investorType, tt.netInterval)
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, resp)
			}
		})
	}
}
