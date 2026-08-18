package repository

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nofendian17/sbterm/apps/api/internal/domain"
	"github.com/nofendian17/sbterm/libs/stockbit"
)

const activityChartRepoBody = `{"data":{"chart_data":[{"type":"TYPE_CHART_VALUE","symbols":["BBRI"],"charts":[{"symbol":"BBRI","chart":[{"date":"2026-08-03","time":"00:00","value":{"raw":"-11939190000","formatted":"(11.9B)"},"datetime_label":"03 Aug"}]}]}]}}`

const activityRepoBody = `{"data":{"broker_activity_transaction":{"brokers_buy":[{"stock_code":"BBCA","broker_code":"YU","type":"BROKER_TYPE_LOCAL","date":"2026-07-14","value":4715906285000,"lot":7425406,"avg_price":6351.0416602135965,"freq":90295,"company_detail":{"icon_url":"https://assets.stockbit.com/logos/companies/BBCA.png","corpaction":{"active":false,"icon":"","text":""},"notation":[]},"nval_trend":[{"date":"2026-08-03","nval":122012707500,"nvol":193528,"nfreq":6035}]}],"brokers_sell":[]},"from":"2026-07-14","to":"2026-07-31","broker_code":"AK, YU, ZP","broker_name":""}}`

const activityHistoricalRepoBody = `{"data":{"date_from":"2026-07-01","date_to":"2026-08-12","symbols":["CUAN"],"broker_codes":["ZP","BK"],"broker_name":"","records":[{"date":"2026-08-12","broker_code":"","trade_activity":{"net_summary":{"avg_price":796.7837407992519,"freq":4947,"lot":-141664,"value":-11740235500},"buy_summary":{"avg_price":786.0812149497357,"freq":1937,"lot":422964,"value":33248405500},"sell_summary":{"avg_price":796.7837407992519,"freq":4947,"lot":564628,"value":44988641000},"foreign_summary":{"foreign_buy":0,"foreign_sell":0,"net_foreign":0},"total_buy_lot":{"amount":422964,"pct":42.82780743464913},"total_sell_lot":{"amount":564628,"pct":57.172192565350876}},"price_activity":{"close_price":"870","return_summary":{"amount":73.21625920074814,"pct":8.415661977097487}}},{"date":"2026-08-11","broker_code":"","trade_activity":{"net_summary":{"avg_price":687.0822704832468,"freq":531,"lot":51824,"value":3522402000},"buy_summary":{"avg_price":687.0822704832468,"freq":531,"lot":127409,"value":8754046500},"sell_summary":{"avg_price":692.1538003572138,"freq":722,"lot":75585,"value":5231644500},"foreign_summary":{"foreign_buy":0,"foreign_sell":0,"net_foreign":0},"total_buy_lot":{"amount":127409,"pct":62.76490930766426},"total_sell_lot":{"amount":75585,"pct":37.23509069233573}},"price_activity":{"close_price":"720","return_summary":{"amount":32.917729516753184,"pct":4.571906877326831}}}],"pagination":{"page":1,"limit":100,"has_next":false,"has_prev":false},"summary":{"group_type":"INTERVAL_TYPE_MONTHLY","data":[{"date_from":"2026-08-01","date_to":"2026-08-12","net_summary":{"avg_price":731.6727956504039,"freq":13735,"lot":-514137,"value":-36687700500}},{"date_from":"2026-07-01","date_to":"2026-07-31","net_summary":{"avg_price":676.1897003304894,"freq":24248,"lot":7726,"value":-1823597000}}]}}}`

func TestActivityRepositoryGetActivityChart(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		body     string
		wantErr  bool
		wantUp   bool
		wantCode int
	}{
		{
			name:   "returns mapped activity chart",
			status: http.StatusOK,
			body:   activityChartRepoBody,
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
				assert.Equal(t, "/order-trade/broker/activity-chart", r.URL.Path)
				assert.Equal(t, []string{"DR", "AK"}, r.URL.Query()["brokers_code"])
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			client := stockbit.New(stockbit.WithBaseURL(srv.URL))
			repo := NewActivityRepository(client)

			got, err := repo.GetActivityChart(context.Background(), []string{"BBRI"}, []string{"DR", "AK"}, "2026-07-01", "2026-08-10", "", "INVESTOR_TYPE_ALL", "BOARD_TYPE_ALL")
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
			require.Len(t, got.ChartData, 1)
			g := got.ChartData[0]
			assert.Equal(t, "TYPE_CHART_VALUE", g.Type)
			assert.Equal(t, []string{"BBRI"}, g.Symbols)
			require.Len(t, g.Charts, 1)
			require.Len(t, g.Charts[0].Chart, 1)
			assert.Equal(t, "(11.9B)", g.Charts[0].Chart[0].Value.Formatted)
			assert.Equal(t, "03 Aug", g.Charts[0].Chart[0].DatetimeLabel)
		})
	}
}

func TestActivityRepositoryGetActivity(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		body     string
		wantErr  bool
		wantUp   bool
		wantCode int
	}{
		{
			name:   "returns mapped activity",
			status: http.StatusOK,
			body:   activityRepoBody,
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
				assert.Equal(t, "/order-trade/broker/activity", r.URL.Path)
				assert.Equal(t, []string{"AK", "DR"}, r.URL.Query()["broker_code"])
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			client := stockbit.New(stockbit.WithBaseURL(srv.URL))
			repo := NewActivityRepository(client)

			got, err := repo.GetActivity(context.Background(), []string{"AK", "DR"}, "TRANSACTION_TYPE_GROSS", "INVESTOR_TYPE_ALL", "MARKET_TYPE_REGULER", 20, 1, "2026-07-14", "2026-07-31", "NET_VAL_PERIOD_7D")
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
			assert.Equal(t, "2026-07-14", got.From)
			assert.Equal(t, "AK, YU, ZP", got.BrokerCode)
			ba := got.BrokerActivityTransaction
			require.Len(t, ba.BrokersBuy, 1)
			b := ba.BrokersBuy[0]
			assert.Equal(t, "BBCA", b.StockCode)
			assert.Equal(t, "YU", b.BrokerCode)
			assert.Equal(t, "BROKER_TYPE_LOCAL", b.Type)
			assert.Equal(t, 4715906285000.0, b.Value)
			assert.Equal(t, 6351.0416602135965, b.AveragePrice)
			assert.False(t, b.CompanyDetail.CorpAction.Active)
			require.Len(t, b.NetValueTrend, 1)
			assert.Equal(t, 193528.0, b.NetValueTrend[0].NVol)
		})
	}
}

func TestActivityRepositoryGetActivityHistorical(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		body     string
		wantErr  bool
		wantUp   bool
		wantCode int
	}{
		{
			name:   "returns mapped activity historical",
			status: http.StatusOK,
			body:   activityHistoricalRepoBody,
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
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			client := stockbit.New(stockbit.WithBaseURL(srv.URL))
			repo := NewActivityRepository(client)

			got, err := repo.GetActivityHistorical(context.Background(), "INTERVAL_DAILY", "2026-07-01", "2026-08-31", []string{"ZP", "BK"}, []string{"CUAN"}, "BOARD_TYPE_REGULAR", "INVESTOR_TYPE_ALL", "INTERVAL_MONTHLY")
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
			assert.Equal(t, "2026-07-01", got.DateFrom)
			assert.Equal(t, []string{"CUAN"}, got.Symbols)
			assert.Equal(t, []string{"ZP", "BK"}, got.BrokerCodes)
			require.Len(t, got.Records, 2)
			rec := got.Records[0]
			assert.Equal(t, "2026-08-12", rec.Date)
			assert.Equal(t, -141664.0, rec.TradeActivity.NetSummary.Lot)
			assert.Equal(t, 0.0, rec.TradeActivity.ForeignSummary.NetForeign)
			assert.Equal(t, 42.82780743464913, rec.TradeActivity.TotalBuyLot.Pct)
			assert.Equal(t, "870", rec.PriceActivity.ClosePrice)
			assert.Equal(t, 8.415661977097487, rec.PriceActivity.ReturnSummary.Pct)
			assert.Equal(t, 1, got.Pagination.Page)
			assert.False(t, got.Pagination.HasNext)
			assert.Equal(t, "INTERVAL_TYPE_MONTHLY", got.Summary.GroupType)
			require.Len(t, got.Summary.Data, 2)
			assert.Equal(t, "2026-07-01", got.Summary.Data[1].DateFrom)
			assert.Equal(t, -1823597000.0, got.Summary.Data[1].NetSummary.Value)
		})
	}
}
