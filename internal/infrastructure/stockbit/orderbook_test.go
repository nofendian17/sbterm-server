package stockbit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// orderBookBody mirrors the real upstream response captured from
// /company-price-feed/v2/orderbook/companies/VKTR (see /tmp/opencode/ob.json),
// trimmed to a few bid/offer levels and key stats.
const orderBookBody = `{"data":{"average":880,"bid":[{"price":"880","que_num":"138","volume":"823200","change_percentage":""},{"price":"875","que_num":"296","volume":"2567800","change_percentage":""}],"change":65,"close":885,"country":"ID","domestic":"81.14","down":"110","exchange":"IDX","fbuy":45454808500,"fnet":24004280500,"foreign":"18.86","frequency":31496,"fsell":21450528000,"high":920,"id":"VKTR-0","lastprice":885,"low":810,"offer":[{"price":"885","que_num":"72","volume":"497200","change_percentage":""}],"open":820,"percentage_change":7.93,"previous":820,"status":"Active","symbol":"VKTR","symbol_2":"VKTR","symbol_3":"VKTR","tradable":true,"unchanged":"1118","up":"265","value":177331201000,"volume":201417800,"corp_action":{"active":false,"icon":"https://assets.stockbit.com/images/corp_action_event_icon.svg","text":"Perusahaan Memiliki Corporate Action"},"notation":[],"uma":false,"has_foreign_bs":true,"iepiev":{"symbol":"","status":"STATUS_UNSPECIFIED"},"market_data":[{"label":"All Market","frequency":{"raw":"31496","formatted":"31.5 K"},"volume":{"raw":"201417800","formatted":"201 M"},"value":{"raw":"177331201000","formatted":"177 B"}}],"name":"VKTR Teknologi Mobilitas Tbk.","icon_url":"https://assets.stockbit.com/logos/companies/VKTR.png","ara":{"value":"1,105","visible":true},"arb":{"value":"755","visible":true},"company_type":"Saham","total_bid_offer":{"bid":{"freq":"2,762","lot":"39,578,900","raw_lot":"39578900","raw_freq":"2762"},"offer":{"freq":"3,885","lot":"53,346,400","raw_lot":"53346400","raw_freq":"3885"},"bid_percent":42.6},"next_ara":{"value":"1,105","visible":true},"next_arb":{"value":"755","visible":true},"autoreject_time_left_in_sec":0,"auto_reject_estimation":[{"value":115100,"type":"AUTO_REJECT_TYPE_POSITIVE"}],"orderbook_active_feature_mobile":"ORDER_BOOK_FEATURE_FOREIGN_BS"}}`

func TestGetOrderBook(t *testing.T) {
	tests := []struct {
		name    string
		opts    []Option
		handler func(t *testing.T, w http.ResponseWriter, r *http.Request)
		check   func(t *testing.T, resp *OrderBookResponse)
	}{
		{
			name: "returns order book",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/company-price-feed/v2/orderbook/companies/VKTR", r.URL.Path)
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(orderBookBody))
			},
			check: func(t *testing.T, resp *OrderBookResponse) {
				d := resp.Data
				assert.Equal(t, "VKTR", d.Symbol)
				assert.Equal(t, 880, d.Average)
				assert.Equal(t, 885, d.LastPrice)
				assert.Equal(t, 7.93, d.PercentageChange)
				assert.True(t, d.Tradable)
				assert.Equal(t, int64(45454808500), d.FBuy)
				assert.Equal(t, int64(24004280500), d.FNet)
				require.Len(t, d.Bid, 2)
				assert.Equal(t, "880", d.Bid[0].Price)
				assert.Equal(t, "823200", d.Bid[0].Volume)
				assert.Equal(t, "138", d.Bid[0].QueNum)
				require.Len(t, d.Offer, 1)
				assert.Equal(t, "885", d.Offer[0].Price)
				assert.False(t, d.CorpAction.Active)
				require.Len(t, d.MarketData, 1)
				assert.Equal(t, "All Market", d.MarketData[0].Label)
				assert.Equal(t, "31.5 K", d.MarketData[0].Frequency.Formatted)
				assert.True(t, d.ARA.Visible)
				assert.Equal(t, "1,105", d.NextARA.Value)
				assert.Equal(t, 42.6, d.TotalBidOffer.BidPercent)
				assert.Equal(t, "2762", d.TotalBidOffer.Bid.RawFreq)
				assert.Equal(t, json.RawMessage(`{"value":115100,"type":"AUTO_REJECT_TYPE_POSITIVE"}`), d.AutoRejectEstimation[0])
				assert.Equal(t, "ORDER_BOOK_FEATURE_FOREIGN_BS", d.OrderbookActiveFeatureMobile)
			},
		},
		{
			name: "uses access token",
			opts: []Option{WithAuthenticator(&stubAuth{token: "at-ok"})},
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "Bearer at-ok", r.Header.Get("Authorization"))
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(orderBookBody))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tt.handler(t, w, r)
			}))
			defer srv.Close()

			opts := append([]Option{WithBaseURL(srv.URL)}, tt.opts...)
			resp, err := New(opts...).GetOrderBook(context.Background(), "VKTR")
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, resp)
			}
		})
	}
}
