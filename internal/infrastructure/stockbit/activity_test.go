package stockbit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// activityBody mirrors the real upstream response captured from
// /order-trade/broker/activity (see /tmp/at2.json).
const activityBody = `{"message":"Successfully loaded Broker Activity data","data":{"broker_activity_transaction":{"brokers_buy":[{"stock_code":"BBCA","broker_code":"YU","type":"BROKER_TYPE_LOCAL","date":"2026-07-14","value":4715906285000,"lot":7425406,"avg_price":6351.0416602135965,"freq":90295,"company_detail":{"icon_url":"https://assets.stockbit.com/logos/companies/BBCA.png","corpaction":{"active":false,"icon":"","text":""},"notation":[]},"nval_trend":[{"date":"2026-08-03","nval":122012707500,"nvol":193528,"nfreq":6035}]}],"brokers_sell":[{"stock_code":"BBCA","broker_code":"YU","type":"BROKER_TYPE_LOCAL","date":"2026-07-14","value":4590083512500,"lot":7241358,"avg_price":6338.705409261633,"freq":110312,"company_detail":{"icon_url":"https://assets.stockbit.com/logos/companies/BBCA.png","corpaction":{"active":false,"icon":"","text":""},"notation":[]},"nval_trend":[{"date":"2026-08-03","nval":122012707500,"nvol":193528,"nfreq":6035}]}]},"from":"2026-07-14","to":"2026-07-31","broker_code":"AK, YU, ZP","broker_name":""}}`

func TestGetActivity(t *testing.T) {
	tests := []struct {
		name            string
		brokerCode      []string
		transactionType string
		investorType    string
		marketBoard     string
		limit, page     int
		from, to        string
		netValPeriod    string
		handler         func(t *testing.T, w http.ResponseWriter, r *http.Request)
		check           func(t *testing.T, resp *ActivityDataResponse)
	}{
		{
			name:            "returns activity with all params",
			brokerCode:      []string{"AK", "ZP", "YU"},
			transactionType: "TRANSACTION_TYPE_GROSS",
			investorType:    "INVESTOR_TYPE_ALL",
			marketBoard:     "MARKET_TYPE_REGULER",
			limit:           20,
			page:            1,
			from:            "2026-07-14",
			to:              "2026-07-31",
			netValPeriod:    "NET_VAL_PERIOD_7D",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/order-trade/broker/activity", r.URL.Path)
				q := r.URL.Query()
				assert.Equal(t, []string{"AK", "ZP", "YU"}, q["broker_code"])
				assert.Equal(t, "TRANSACTION_TYPE_GROSS", q.Get("transaction_type"))
				assert.Equal(t, "INVESTOR_TYPE_ALL", q.Get("investor_type"))
				assert.Equal(t, "20", q.Get("limit"))
				assert.Equal(t, "MARKET_TYPE_REGULER", q.Get("market_board"))
				assert.Equal(t, "1", q.Get("page"))
				assert.Equal(t, "2026-07-14", q.Get("from"))
				assert.Equal(t, "2026-07-31", q.Get("to"))
				assert.Equal(t, "NET_VAL_PERIOD_7D", q.Get("net_val_period"))
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(activityBody))
			},
			check: func(t *testing.T, resp *ActivityDataResponse) {
				d := resp.Data
				assert.Equal(t, "2026-07-14", d.From)
				assert.Equal(t, "2026-07-31", d.To)
				assert.Equal(t, "AK, YU, ZP", d.BrokerCode)
				assert.Equal(t, "", d.BrokerName)
				bat := d.BrokerActivityTransaction
				require.Len(t, bat.BrokersBuy, 1)
				require.Len(t, bat.BrokersSell, 1)
				b := bat.BrokersBuy[0]
				assert.Equal(t, "BBCA", b.StockCode)
				assert.Equal(t, "YU", b.BrokerCode)
				assert.Equal(t, "BROKER_TYPE_LOCAL", b.Type)
				assert.Equal(t, "2026-07-14", b.Date)
				assert.Equal(t, 4715906285000.0, b.Value)
				assert.Equal(t, 7425406.0, b.Lot)
				assert.Equal(t, 6351.0416602135965, b.AveragePrice)
				assert.Equal(t, 90295.0, b.Frequency)
				assert.Equal(t, "https://assets.stockbit.com/logos/companies/BBCA.png", b.CompanyDetail.IconURL)
				assert.False(t, b.CompanyDetail.CorpAction.Active)
				assert.Empty(t, b.CompanyDetail.Notation)
				require.Len(t, b.NetValueTrend, 1)
				assert.Equal(t, "2026-08-03", b.NetValueTrend[0].Date)
				assert.Equal(t, 122012707500.0, b.NetValueTrend[0].NVal)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tt.handler(t, w, r)
			}))
			defer srv.Close()

			resp, err := New(WithBaseURL(srv.URL)).GetActivity(context.Background(), tt.brokerCode, tt.transactionType, tt.investorType, tt.marketBoard, tt.limit, tt.page, tt.from, tt.to, tt.netValPeriod)
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, resp)
			}
		})
	}
}
