package stockbit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// brokerTopBody mirrors the real upstream response captured from
// /order-trade/broker/top (see /tmp/tb1.json).
const brokerTopBody = `{"message":"Successfully get top broker","data":{"date":{"from":"2026-08-11","to":"2026-08-11","idx":"2026-08-11"},"list":[{"code":"XL","name":"Stockbit Sekuritas Digital","investor_type":"INVESTOR_TYPE_UNSPECIFIED","total_value":"3954882296950","net_value":"31002582100","buy_value":"1992942439525","sell_value":"1961939857425","total_volume":"16818162166","total_frequency":"1431582","group":"BROKER_GROUP_LOCAL"},{"code":"AK","name":"UBS Sekuritas Indonesia","investor_type":"INVESTOR_TYPE_UNSPECIFIED","total_value":"2972454430720","net_value":"-364958830900","buy_value":"1303747799910","sell_value":"1668706630810","total_volume":"4282349294","total_frequency":"271644","group":"BROKER_GROUP_FOREIGN"}]}}`

func TestGetBrokerTop(t *testing.T) {
	tests := []struct {
		name       string
		sort       string
		order      string
		period     string
		marketType string
		eodOnly    bool
		handler    func(t *testing.T, w http.ResponseWriter, r *http.Request)
		check      func(t *testing.T, resp *BrokerTopResponse)
	}{
		{
			name:       "returns broker top with all params",
			sort:       "TB_SORT_BY_TOTAL_VALUE",
			order:      "ORDER_BY_DESC",
			period:     "TB_PERIOD_LAST_1_DAY",
			marketType: "MARKET_TYPE_ALL",
			eodOnly:    true,
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/order-trade/broker/top", r.URL.Path)
				q := r.URL.Query()
				assert.Equal(t, "TB_SORT_BY_TOTAL_VALUE", q.Get("sort"))
				assert.Equal(t, "ORDER_BY_DESC", q.Get("order"))
				assert.Equal(t, "TB_PERIOD_LAST_1_DAY", q.Get("period"))
				assert.Equal(t, "MARKET_TYPE_ALL", q.Get("market_type"))
				assert.Equal(t, "true", q.Get("eod_only"))
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(brokerTopBody))
			},
			check: func(t *testing.T, resp *BrokerTopResponse) {
				d := resp.Data
				assert.Equal(t, "2026-08-11", d.Date.From)
				assert.Equal(t, "2026-08-11", d.Date.To)
				assert.Equal(t, "2026-08-11", d.Date.Idx)
				require.Len(t, d.List, 2)
				first := d.List[0]
				assert.Equal(t, "XL", first.Code)
				assert.Equal(t, "Stockbit Sekuritas Digital", first.Name)
				assert.Equal(t, "INVESTOR_TYPE_UNSPECIFIED", first.InvestorType)
				assert.Equal(t, "3954882296950", first.TotalValue)
				assert.Equal(t, "31002582100", first.NetValue)
				assert.Equal(t, "1992942439525", first.BuyValue)
				assert.Equal(t, "1961939857425", first.SellValue)
				assert.Equal(t, "16818162166", first.TotalVolume)
				assert.Equal(t, "1431582", first.TotalFrequency)
				assert.Equal(t, "BROKER_GROUP_LOCAL", first.Group)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tt.handler(t, w, r)
			}))
			defer srv.Close()

			resp, err := New(WithBaseURL(srv.URL)).GetBrokerTop(context.Background(), tt.sort, tt.order, tt.period, tt.marketType, tt.eodOnly)
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, resp)
			}
		})
	}
}
