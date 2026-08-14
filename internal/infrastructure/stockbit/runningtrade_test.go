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

// runningTradeBody mirrors the real upstream response captured from
// /order-trade/running-trade/chart/DSSA (see /tmp/rt_dssa.json): a price series
// plus two broker groups (TYPE_CHART_VALUE and TYPE_CHART_VOLUME).
const runningTradeBody = `{"message":"Successfully loaded running trade chart","data":{"from":"2026-08-02","to":"2026-08-11","data_last_updated":"2026-08-11T00:00:00Z","price_chart_data":[{"date":"2026-08-03","time":"00:00","value":{"raw":"835","formatted":"835"},"datetime_label":"03 Aug","open":{"raw":"850","formatted":"850"},"high":{"raw":"875","formatted":"875"},"low":{"raw":"830","formatted":"830"}}],"broker_chart_data":[{"type":"TYPE_CHART_VALUE","brokers":["XL","BK","AK","CC","YU"],"charts":[{"broker_code":"AK","chart":[{"date":"2026-08-03","time":"00:00","value":{"raw":"-11939190000","formatted":"(11.9B)"},"datetime_label":"03 Aug","open":null,"high":null,"low":null}]}]},{"type":"TYPE_CHART_VOLUME","brokers":["XL","BK","AK","CC","YU"],"charts":[{"broker_code":"XL","chart":[{"date":"2026-08-03","time":"00:00","value":{"raw":"-46272","formatted":"(46.3K)"},"datetime_label":"03 Aug","open":null,"high":null,"low":null}]}]}],"date_session_info":"11 Aug 2026"}}`

// runningTradeFeedBody mirrors the real upstream response captured from
// /order-trade/running-trade (see /tmp/opencode/rt_multi.json).
const runningTradeFeedBody = `{"message":"Successfully loaded running trade data","data":{"is_open_market":false,"running_trade":[{"id":"4760187264","time":"08:58:00","action":"buy","code":"BBCA","price":"6,300","change":"-1.18%","lot":"1","is_broker_exists":true,"buyer":"XL [D]","seller":"BK [F]","trade_number":"17797","buyer_type":"BROKER_TYPE_LOCAL","seller_type":"BROKER_TYPE_FOREIGN","market_board":"RG","buy_order_number":"85260","sell_order_number":"13070","group_order_number":"13070","value":{"raw":630000,"formatted":"630.0K"}}]}}`

func TestGetRunningTrade(t *testing.T) {
	tests := []struct {
		name        string
		opts        []Option
		symbol      string
		sort        string
		orderBy     string
		date        string
		limit       int
		tradeNumber int64
		handler     func(t *testing.T, w http.ResponseWriter, r *http.Request)
		check       func(t *testing.T, resp *RunningTradeFeedResponse)
	}{
		{
			name:        "returns running trade feed with all params",
			symbol:      "BBCA",
			sort:        "ASC",
			orderBy:     "RUNNING_TRADE_ORDER_BY_TIME",
			date:        "2026-08-13",
			limit:       80,
			tradeNumber: 17796,
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/order-trade/running-trade", r.URL.Path)
				q := r.URL.Query()
				assert.Equal(t, "BBCA", q.Get("symbols[]"))
				assert.Equal(t, "ASC", q.Get("sort"))
				assert.Equal(t, "RUNNING_TRADE_ORDER_BY_TIME", q.Get("order_by"))
				assert.Equal(t, "2026-08-13", q.Get("date"))
				assert.Equal(t, "80", q.Get("limit"))
				assert.Equal(t, "17796", q.Get("trade_number"))
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(runningTradeFeedBody))
			},
			check: func(t *testing.T, resp *RunningTradeFeedResponse) {
				assert.False(t, resp.Data.IsOpenMarket)
				require.Len(t, resp.Data.RunningTrade, 1)
				item := resp.Data.RunningTrade[0]
				assert.Equal(t, "4760187264", item.ID)
				assert.Equal(t, "08:58:00", item.Time)
				assert.Equal(t, "buy", item.Action)
				assert.Equal(t, "BBCA", item.Code)
				assert.Equal(t, "6,300", item.Price)
				assert.Equal(t, "-1.18%", item.Change)
				assert.Equal(t, "XL [D]", item.Buyer)
				assert.Equal(t, "BK [F]", item.Seller)
				assert.Equal(t, "BROKER_TYPE_LOCAL", item.BuyerType)
				assert.Equal(t, "BROKER_TYPE_FOREIGN", item.SellerType)
				assert.Equal(t, "RG", item.MarketBoard)
				assert.Equal(t, "17797", item.TradeNumber)
				assert.Equal(t, json.Number("630000"), item.Value.Raw)
				assert.Equal(t, "630.0K", item.Value.Formatted)
			},
		},
		{
			name:   "omits optional params when empty",
			symbol: "BBCA",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				q := r.URL.Query()
				assert.Equal(t, "BBCA", q.Get("symbols[]"))
				assert.Equal(t, "", q.Get("sort"))
				assert.Equal(t, "", q.Get("order_by"))
				assert.Equal(t, "", q.Get("date"))
				assert.Equal(t, "", q.Get("limit"))
				assert.Equal(t, "", q.Get("trade_number"))
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(runningTradeFeedBody))
			},
			check: func(t *testing.T, resp *RunningTradeFeedResponse) {
				require.Len(t, resp.Data.RunningTrade, 1)
			},
		},
		{
			name:   "uses access token",
			symbol: "BBCA",
			opts:   []Option{WithAuthenticator(&stubAuth{token: "at-ok"})},
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "Bearer at-ok", r.Header.Get("Authorization"))
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(runningTradeFeedBody))
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
			resp, err := New(opts...).GetRunningTrade(
				context.Background(), tt.symbol, tt.sort, tt.orderBy, tt.date, tt.limit, tt.tradeNumber)
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, resp)
			}
		})
	}
}

func TestGetRunningTradeChart(t *testing.T) {
	tests := []struct {
		name         string
		symbol       string
		brokerCodes  []string
		from, to     string
		investorType string
		marketBoard  string
		period       string
		handler      func(t *testing.T, w http.ResponseWriter, r *http.Request)
		check        func(t *testing.T, resp *RunningTradeResponse)
	}{
		{
			name:         "returns running trade chart with from/to",
			symbol:       "DSSA",
			brokerCodes:  []string{"DR", "AK", "DH", "ZP", "HP"},
			from:         "2026-07-01",
			to:           "2026-08-10",
			investorType: "INVESTOR_TYPE_ALL",
			marketBoard:  "BOARD_TYPE_ALL",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/order-trade/running-trade/chart/DSSA", r.URL.Path)
				assert.Equal(t, "2026-07-01", r.URL.Query().Get("from"))
				assert.Equal(t, "2026-08-10", r.URL.Query().Get("to"))
				assert.Equal(t, []string{"DR", "AK", "DH", "ZP", "HP"}, r.URL.Query()["broker_code"])
				assert.Equal(t, "INVESTOR_TYPE_ALL", r.URL.Query().Get("investor_type"))
				assert.Equal(t, "BOARD_TYPE_ALL", r.URL.Query().Get("market_board"))
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(runningTradeBody))
			},
			check: func(t *testing.T, resp *RunningTradeResponse) {
				d := resp.Data
				assert.Equal(t, "2026-08-02", d.From)
				assert.Equal(t, "2026-08-11", d.To)
				assert.Equal(t, "11 Aug 2026", d.DateSessionInfo)
				require.Len(t, d.PriceChartData, 1)
				p := d.PriceChartData[0]
				assert.Equal(t, "835", p.Value.Raw)
				assert.Equal(t, "850", p.Open.Raw)
				require.Len(t, d.BrokerChartData, 2)
				assert.Equal(t, "TYPE_CHART_VALUE", d.BrokerChartData[0].Type)
				require.Len(t, d.BrokerChartData[0].Charts, 1)
				assert.Equal(t, "AK", d.BrokerChartData[0].Charts[0].BrokerCode)
				bp := d.BrokerChartData[0].Charts[0].Chart[0]
				assert.Equal(t, "(11.9B)", bp.Value.Formatted)
				assert.Nil(t, bp.Open)
			},
		},
		{
			name:         "from/to win over period when both supplied",
			symbol:       "DSSA",
			brokerCodes:  []string{"DR"},
			from:         "2026-07-01",
			to:           "2026-08-10",
			investorType: "INVESTOR_TYPE_ALL",
			marketBoard:  "BOARD_TYPE_ALL",
			period:       "RT_PERIOD_LAST_7_DAYS",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/order-trade/running-trade/chart/DSSA", r.URL.Path)
				assert.Equal(t, "2026-07-01", r.URL.Query().Get("from"))
				assert.Equal(t, "2026-08-10", r.URL.Query().Get("to"))
				assert.Equal(t, "", r.URL.Query().Get("period"))
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(runningTradeBody))
			},
			check: func(t *testing.T, resp *RunningTradeResponse) {
				require.Len(t, resp.Data.PriceChartData, 1)
			},
		},
		{
			name:         "sends period when from/to empty and omits from/to",
			symbol:       "DSSA",
			brokerCodes:  []string{"DR"},
			investorType: "INVESTOR_TYPE_ALL",
			marketBoard:  "BOARD_TYPE_ALL",
			period:       "RT_PERIOD_LAST_1_DAY",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/order-trade/running-trade/chart/DSSA", r.URL.Path)
				assert.Equal(t, "RT_PERIOD_LAST_1_DAY", r.URL.Query().Get("period"))
				assert.Equal(t, "", r.URL.Query().Get("from"))
				assert.Equal(t, "", r.URL.Query().Get("to"))
				assert.Equal(t, []string{"DR"}, r.URL.Query()["broker_code"])
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(runningTradeBody))
			},
			check: func(t *testing.T, resp *RunningTradeResponse) {
				require.Len(t, resp.Data.BrokerChartData, 2)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tt.handler(t, w, r)
			}))
			defer srv.Close()

			resp, err := New(WithBaseURL(srv.URL)).GetRunningTradeChart(
				context.Background(), tt.symbol, tt.brokerCodes, tt.from, tt.to, tt.investorType, tt.marketBoard, tt.period)
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, resp)
			}
		})
	}
}
