package repository

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nofendian17/sbterm-server/internal/domain"
	"github.com/nofendian17/sbterm-server/internal/infrastructure/stockbit"
)

const activityChartRepoBody = `{"data":{"chart_data":[{"type":"TYPE_CHART_VALUE","symbols":["BBRI"],"charts":[{"symbol":"BBRI","chart":[{"date":"2026-08-03","time":"00:00","value":{"raw":"-11939190000","formatted":"(11.9B)"},"datetime_label":"03 Aug"}]}]}]}}`

const activityRepoBody = `{"data":{"broker_activity_transaction":{"brokers_buy":[{"stock_code":"BBCA","broker_code":"YU","type":"BROKER_TYPE_LOCAL","date":"2026-07-14","value":4715906285000,"lot":7425406,"avg_price":6351.0416602135965,"freq":90295,"company_detail":{"icon_url":"https://assets.stockbit.com/logos/companies/BBCA.png","corpaction":{"active":false,"icon":"","text":""},"notation":[]},"nval_trend":[{"date":"2026-08-03","nval":122012707500,"nvol":193528,"nfreq":6035}]}],"brokers_sell":[]},"from":"2026-07-14","to":"2026-07-31","broker_code":"AK, YU, ZP","broker_name":""}}`

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
