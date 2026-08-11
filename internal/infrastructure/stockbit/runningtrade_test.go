package stockbit

import (
	"context"
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
