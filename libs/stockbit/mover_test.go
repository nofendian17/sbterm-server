package stockbit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const moverBody = `{"message":"Successfully get list market mover","data":{"mover_list":[{"stock_detail":{"code":"VOKS","name":"Voksel Electric Tbk.","icon_url":"https://assets.stockbit.com/logos/companies/VOKS.png","has_uma":false,"notations":[],"corpaction":{"active":false,"icon_url":"https://assets.stockbit.com/images/corp_action_event_icon.svg","text":"Perusahaan Memiliki Corporate Action"}},"price":270,"change":{"value":70,"percentage":35},"value":{"raw":4795797800,"formatted":"4.80 B"},"volume":{"raw":183517,"formatted":"183.52 K"},"frequency":{"raw":2743,"formatted":"2.74 K"},"net_foreign_buy":{"raw":0,"formatted":"-"},"net_foreign_sell":{"raw":164914400,"formatted":"164.91 M"},"net_buy":{"raw":3015890400,"formatted":""},"net_sell":{"raw":1779907400,"formatted":""},"iepiev_detail":{"iep":{"raw":0,"formatted":"0"},"iev":{"raw":0,"formatted":"-"},"ieval":{"raw":0,"formatted":"-"},"iep_change":{"raw":0,"formatted":"0.00%"},"iep_change_prev":{"raw":0,"formatted":"0.00%"},"iep_price_diff":{"raw":0,"formatted":"0"},"iep_prev_price_diff":{"raw":0,"formatted":"0"}},"big_money_net_value":{"raw":0,"formatted":"-"},"buy_value_percentage":0,"sell_value_percentage":0,"big_money_buy_value_percentage":0,"big_money_sell_value_percentage":0,"bid_percent":0,"catalog_detail":{"code":"","name":"","icon_url":"","up_count":0,"down_count":0,"unchanged_count":0,"stock_count":0,"catalog_id":"0"},"market_cap":null}]}}`

func TestGetMarketMover(t *testing.T) {
	tests := []struct {
		name    string
		req     MarketMoverRequest
		opts    []Option
		handler func(t *testing.T, w http.ResponseWriter, r *http.Request)
		check   func(t *testing.T, resp *MarketMoverResponse)
	}{
		{
			name: "sends mover type and repeated board filters",
			req: MarketMoverRequest{
				MoverType:    MoverTypeTopGainer,
				FilterStocks: []FilterStocks{FilterStocksMainBoard, FilterStocksDevelopmentBoard},
			},
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, marketMoverPath, r.URL.Path)
				assert.Equal(t, "MOVER_TYPE_TOP_GAINER", r.URL.Query().Get("mover_type"))
				assert.Equal(t, []string{"FILTER_STOCKS_TYPE_MAIN_BOARD", "FILTER_STOCKS_TYPE_DEVELOPMENT_BOARD"}, r.URL.Query()["filter_stocks"])
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(moverBody))
			},
			check: func(t *testing.T, resp *MarketMoverResponse) {
				require.Len(t, resp.Data.MoverList, 1)
				m := resp.Data.MoverList[0]
				assert.Equal(t, "VOKS", m.StockDetail.Code)
				assert.Equal(t, 270.0, m.Price)
				assert.Equal(t, 35.0, m.Change.Percentage)
				assert.Equal(t, 4795797800.0, m.Value.Raw)
				assert.Equal(t, "4.80 B", m.Value.Formatted)
				assert.Equal(t, 164914400.0, m.NetForeignSell.Raw)
				assert.Nil(t, m.MarketCap)
			},
		},
		{
			name: "uses access token",
			req:  MarketMoverRequest{MoverType: MoverTypeTopGainer},
			opts: []Option{WithAuthenticator(&stubAuth{token: "at-ok"})},
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "Bearer at-ok", r.Header.Get("Authorization"))
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(moverBody))
			},
		},
		{
			name: "omits empty params",
			req:  MarketMoverRequest{},
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Empty(t, r.URL.RawQuery)
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(moverBody))
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
			resp, err := New(opts...).GetMarketMover(context.Background(), tt.req)
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, resp)
			}
		})
	}
}

func TestMarketMoverQueryEncoding(t *testing.T) {
	q := url.Values{}
	q.Set("mover_type", string(MoverTypeTopGainer))
	q.Add("filter_stocks", string(FilterStocksMainBoard))
	q.Add("filter_stocks", string(FilterStocksDevelopmentBoard))
	assert.Equal(t,
		"filter_stocks=FILTER_STOCKS_TYPE_MAIN_BOARD&filter_stocks=FILTER_STOCKS_TYPE_DEVELOPMENT_BOARD&mover_type=MOVER_TYPE_TOP_GAINER",
		q.Encode())
}
