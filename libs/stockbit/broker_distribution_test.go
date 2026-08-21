package stockbit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const brokerDistributionBody = `{"message":"Successfully loaded Broker Distribution data","data":{"date_info":"2026-08-19","by_value":{"top_broker_buy":[{"detail":{"code":"MG","type":"Lokal","amount":683315577200},"distribute_to":[{"code":"XL","type":"Lokal","amount":96841421100},{"code":"CC","type":"Pemerintah","amount":70343513200}]}],"top_broker_sell":[{"detail":{"code":"ZP","type":"Asing","amount":-123},"distribute_to":[]}]},"by_volume":{"top_broker_buy":[],"top_broker_sell":[]},"start_date":"2026-08-02","end_date":"2026-08-19"}}`

func TestGetBrokerDistribution(t *testing.T) {
	tests := []struct {
		name    string
		date    string
		from    string
		to      string
		handler func(t *testing.T, w http.ResponseWriter, r *http.Request)
		check   func(t *testing.T, resp *BrokerDistributionResponse)
		wantErr bool
	}{
		{
			name: "sends symbol, enums and range",
			from: "2026-08-02",
			to:   "2026-08-19",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, brokerDistributionPath, r.URL.Path)
				q := r.URL.Query()
				assert.Equal(t, "BUMI", q.Get("symbol"))
				assert.Equal(t, "INVESTOR_TYPE_FOREIGN", q.Get("investor_type"))
				assert.Equal(t, "MARKET_TYPE_REGULER", q.Get("market_board"))
				assert.Equal(t, "BROKER_DISTRIBUTION_DATA_TYPE_VOLUME", q.Get("data_type"))
				assert.Equal(t, "2026-08-02", q.Get("from"))
				assert.Equal(t, "2026-08-19", q.Get("to"))
				assert.Empty(t, q.Get("date"))
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(brokerDistributionBody))
			},
			check: func(t *testing.T, resp *BrokerDistributionResponse) {
				d := resp.Data
				assert.Equal(t, "2026-08-19", d.DateInfo)
				assert.Equal(t, "2026-08-02", d.StartDate)
				require.Len(t, d.ByValue.TopBrokerBuy, 1)
				e := d.ByValue.TopBrokerBuy[0]
				assert.Equal(t, "MG", e.Detail.Code)
				assert.Equal(t, int64(683315577200), e.Detail.Amount)
				require.Len(t, e.DistributeTo, 2)
				assert.Equal(t, "Pemerintah", e.DistributeTo[1].Type)
				require.Len(t, d.ByValue.TopBrokerSell, 1)
				assert.Empty(t, d.ByVolume.TopBrokerBuy)
			},
		},
		{
			name: "sends date alone and omits range",
			date: "2026-08-19",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				q := r.URL.Query()
				assert.Equal(t, "2026-08-19", q.Get("date"))
				assert.Empty(t, q.Get("from"))
				assert.Empty(t, q.Get("to"))
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(brokerDistributionBody))
			},
			check: func(t *testing.T, resp *BrokerDistributionResponse) {},
		},
		{
			name: "propagates upstream error",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"message":"Invalid date","error_type":"INVALID_PARAMETER"}`))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tt.handler(t, w, r)
			}))
			defer srv.Close()

			resp, err := New(WithBaseURL(srv.URL)).GetBrokerDistribution(context.Background(),
				"BUMI", "INVESTOR_TYPE_FOREIGN", "MARKET_TYPE_REGULER", "BROKER_DISTRIBUTION_DATA_TYPE_VOLUME", tt.date, tt.from, tt.to)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			tt.check(t, resp)
		})
	}
}
