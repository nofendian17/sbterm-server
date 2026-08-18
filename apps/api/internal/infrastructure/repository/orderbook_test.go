package repository

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/libs/stockbit"
)

const orderBookRepoBody = `{"data":{"average":880,"bid":[{"price":"880","que_num":"138","volume":"823200","change_percentage":""}],"change":65,"close":885,"country":"ID","domestic":"81.14","down":"110","exchange":"IDX","fbuy":45454808500,"fnet":24004280500,"foreign":"18.86","frequency":31496,"fsell":21450528000,"high":920,"id":"VKTR-0","lastprice":885,"low":810,"offer":[{"price":"885","que_num":"72","volume":"497200","change_percentage":""}],"open":820,"percentage_change":7.93,"previous":820,"status":"Active","symbol":"VKTR","symbol_2":"VKTR","symbol_3":"VKTR","tradable":true,"unchanged":"1118","up":"265","value":177331201000,"volume":201417800,"corp_action":{"active":false,"icon":"https://assets.stockbit.com/images/corp_action_event_icon.svg","text":"Perusahaan Memiliki Corporate Action"},"notation":[],"uma":false,"has_foreign_bs":true,"iepiev":{"symbol":"","status":"STATUS_UNSPECIFIED"},"market_data":[{"label":"All Market","frequency":{"raw":"31496","formatted":"31.5 K"},"volume":{"raw":"201417800","formatted":"201 M"},"value":{"raw":"177331201000","formatted":"177 B"}}],"name":"VKTR Teknologi Mobilitas Tbk.","icon_url":"https://assets.stockbit.com/logos/companies/VKTR.png","ara":{"value":"1,105","visible":true},"arb":{"value":"755","visible":true},"company_type":"Saham","total_bid_offer":{"bid":{"freq":"2,762","lot":"39,578,900","raw_lot":"39578900","raw_freq":"2762"},"offer":{"freq":"3,885","lot":"53,346,400","raw_lot":"53346400","raw_freq":"3885"},"bid_percent":42.6},"next_ara":{"value":"1,105","visible":true},"next_arb":{"value":"755","visible":true},"autoreject_time_left_in_sec":0,"auto_reject_estimation":[{"value":115100,"type":"AUTO_REJECT_TYPE_POSITIVE"}],"orderbook_active_feature_mobile":"ORDER_BOOK_FEATURE_FOREIGN_BS"}}`

func TestOrderBookRepositoryGetOrderBook(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		body     string
		wantErr  bool
		wantUp   bool
		wantCode int
	}{
		{
			name:   "returns mapped order book",
			status: http.StatusOK,
			body:   orderBookRepoBody,
		},
		{
			name:    "propagates upstream error",
			status:  http.StatusInternalServerError,
			body:    `{"message":"boom"}`,
			wantErr: true,
		},
		{
			name:     "translates upstream 400 into domain error",
			status:   http.StatusBadRequest,
			body:     `{"message":"Please check your request"}`,
			wantErr:  true,
			wantUp:   true,
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/company-price-feed/v2/orderbook/companies/VKTR", r.URL.Path)
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			client := stockbit.New(stockbit.WithBaseURL(srv.URL))
			repo := NewOrderBookRepository(client)

			got, err := repo.GetOrderBook(context.Background(), "VKTR")
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantUp {
					var up *domain.UpstreamError
					require.ErrorAs(t, err, &up)
					assert.Equal(t, tt.wantCode, up.Status)
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "VKTR", got.Symbol)
			assert.Equal(t, 880, got.Average)
			assert.Equal(t, 885, got.LastPrice)
			assert.Equal(t, 7.93, got.PercentageChange)
			assert.True(t, got.Tradable)
			assert.Equal(t, int64(177331201000), got.Value)
			require.Len(t, got.Bid, 1)
			assert.Equal(t, "880", got.Bid[0].Price)
			assert.Equal(t, "823200", got.Bid[0].Volume)
			require.Len(t, got.Offer, 1)
			assert.Equal(t, "885", got.Offer[0].Price)
			require.Len(t, got.MarketData, 1)
			assert.Equal(t, "31.5 K", got.MarketData[0].Frequency.Formatted)
			assert.Equal(t, 42.6, got.TotalBidOffer.BidPercent)
			assert.Equal(t, "2762", got.TotalBidOffer.Bid.RawFreq)
			assert.Equal(t, json.RawMessage(`{"symbol":"","status":"STATUS_UNSPECIFIED"}`), got.IEPIEV)
			assert.Equal(t, json.RawMessage(`{"value":115100,"type":"AUTO_REJECT_TYPE_POSITIVE"}`), got.AutoRejectEstimation[0])
		})
	}
}
